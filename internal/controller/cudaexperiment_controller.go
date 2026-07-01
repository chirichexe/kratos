/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gpuv1alpha1 "github.com/chirichexe/kratos/api/v1alpha1"
)

const (
	executionJobConditionType       = "ExecutionJobCreated"
	failedConditionType             = "Failed"
	profileSummaryConfigMapKey      = "summary.json"
	profilingCompletedConditionType = "ProfilingCompleted"
	profilingPendingConditionType   = "ProfilingPending"
	profilingRunningConditionType   = "ProfilingRunning"
	defaultRuntimeClassName         = "nvidia"
	defaultNsightComputeImage       = "kratos-nsight-compute-poc:latest"
	defaultWorkloadExecutable       = "/cuda-samples/vectorAdd"
	sharedWorkloadVolumeName        = "workload"
	sharedWorkloadMountPath         = "/shared"
)

var (
	errInvalidProfileSummary = errors.New("invalid profile summary")
	errProfileSummaryMissing = errors.New("profile summary missing")
)

// CUDAExperimentReconciler reconciles a CUDAExperiment object
type CUDAExperimentReconciler struct {
	client.Client
	Scheme             *runtime.Scheme
	NsightComputeImage string
}

// +kubebuilder:rbac:groups=gpu.scheduler.io,resources=cudaexperiments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gpu.scheduler.io,resources=cudaexperiments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gpu.scheduler.io,resources=cudaexperiments/finalizers,verbs=update
// +kubebuilder:rbac:groups=gpu.scheduler.io,resources=workloadprofiles,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=gpu.scheduler.io,resources=workloadprofiles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.24.1/pkg/reconcile
func (r *CUDAExperimentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var experiment gpuv1alpha1.CUDAExperiment
	if err := r.Get(ctx, req.NamespacedName, &experiment); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !experiment.Spec.ProfilingEnabled {
		if err := r.ensureExecutionJob(ctx, &experiment); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, r.setExecutionJobCreatedCondition(ctx, &experiment)
	}

	profileExists, err := r.workloadProfileExists(ctx, &experiment)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !profileExists {
		profilingJob, created, err := r.ensureProfilingJob(ctx, &experiment)
		if err != nil {
			return ctrl.Result{}, err
		}
		if created {
			return ctrl.Result{}, r.setProfilingPendingCondition(ctx, &experiment)
		}
		if isJobFailed(profilingJob) {
			return ctrl.Result{}, r.setFailedCondition(
				ctx,
				&experiment,
				"ProfilingJobFailed",
				fmt.Sprintf("Profiling Job %q failed", profilingJobName(&experiment)),
			)
		}
		if isJobComplete(profilingJob) {
			if err := r.createOrUpdateWorkloadProfileFromSummary(ctx, &experiment); err != nil {
				if errors.Is(err, errProfileSummaryMissing) {
					return ctrl.Result{}, r.setFailedCondition(
						ctx,
						&experiment,
						"ProfileSummaryMissing",
						fmt.Sprintf("Profile summary ConfigMap %q or key %q is missing", profileSummaryConfigMapName(&experiment), profileSummaryConfigMapKey),
					)
				}
				if errors.Is(err, errInvalidProfileSummary) {
					return ctrl.Result{}, r.setFailedCondition(
						ctx,
						&experiment,
						"InvalidProfileSummary",
						fmt.Sprintf("Profile summary ConfigMap %q contains invalid %q", profileSummaryConfigMapName(&experiment), profileSummaryConfigMapKey),
					)
				}
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, r.setProfilingCompletedCondition(ctx, &experiment)
		}
		return ctrl.Result{}, r.setProfilingRunningCondition(ctx, &experiment)
	}

	if err := r.ensureExecutionJob(ctx, &experiment); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, r.setExecutionJobCreatedCondition(ctx, &experiment)
}

func (r *CUDAExperimentReconciler) ensureExecutionJob(ctx context.Context, experiment *gpuv1alpha1.CUDAExperiment) error {
	jobName := executionJobName(experiment)
	var job batchv1.Job
	err := r.Get(ctx, types.NamespacedName{Name: jobName, Namespace: experiment.Namespace}, &job)
	if apierrors.IsNotFound(err) {
		job = r.executionJobForExperiment(experiment)
		if err := ctrl.SetControllerReference(experiment, &job, r.Scheme); err != nil {
			return err
		}
		if err := r.Create(ctx, &job); err != nil {
			return err
		}
		log := ctrl.LoggerFrom(ctx)
		log.Info("Created Job", "name", job.Name, "namespace", job.Namespace)
		return nil
	}
	return err
}

func (r *CUDAExperimentReconciler) ensureProfilingJob(
	ctx context.Context,
	experiment *gpuv1alpha1.CUDAExperiment,
) (*batchv1.Job, bool, error) {
	jobName := profilingJobName(experiment)
	job := &batchv1.Job{}
	err := r.Get(ctx, types.NamespacedName{Name: jobName, Namespace: experiment.Namespace}, job)
	if apierrors.IsNotFound(err) {
		newJob := r.profilingJobForExperiment(experiment)
		if err := ctrl.SetControllerReference(experiment, &newJob, r.Scheme); err != nil {
			return nil, false, err
		}
		if err := r.Create(ctx, &newJob); err != nil {
			return nil, false, err
		}
		log := ctrl.LoggerFrom(ctx)
		log.Info("Created Job", "name", newJob.Name, "namespace", newJob.Namespace)
		return &newJob, true, nil
	}
	if err != nil {
		return nil, false, err
	}
	return job, false, nil
}

func (r *CUDAExperimentReconciler) workloadProfileExists(ctx context.Context, experiment *gpuv1alpha1.CUDAExperiment) (bool, error) {
	var profile gpuv1alpha1.WorkloadProfile
	err := r.Get(ctx, types.NamespacedName{Name: workloadProfileName(experiment), Namespace: experiment.Namespace}, &profile)
	if apierrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *CUDAExperimentReconciler) createOrUpdateWorkloadProfileFromSummary(
	ctx context.Context,
	experiment *gpuv1alpha1.CUDAExperiment,
) error {
	summary, err := r.profileSummary(ctx, experiment)
	if err != nil {
		return err
	}

	profileName := workloadProfileName(experiment)
	profile := &gpuv1alpha1.WorkloadProfile{}
	err = r.Get(ctx, types.NamespacedName{Name: profileName, Namespace: experiment.Namespace}, profile)
	if apierrors.IsNotFound(err) {
		profile = &gpuv1alpha1.WorkloadProfile{
			ObjectMeta: metav1.ObjectMeta{
				Name:      profileName,
				Namespace: experiment.Namespace,
			},
		}
		setWorkloadProfileSpec(profile, experiment)
		if err := ctrl.SetControllerReference(experiment, profile, r.Scheme); err != nil {
			return err
		}
		if err := r.Create(ctx, profile); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		setWorkloadProfileSpec(profile, experiment)
		if err := ctrl.SetControllerReference(experiment, profile, r.Scheme); err != nil {
			return err
		}
		if err := r.Update(ctx, profile); err != nil {
			return err
		}
	}

	profile.Status.BoundType = summary.BoundType
	profile.Status.Metrics = summary.Metrics
	if err := r.Status().Update(ctx, profile); err != nil {
		return err
	}

	log := ctrl.LoggerFrom(ctx)
	log.Info("Updated WorkloadProfile from profile summary", "name", profile.Name, "namespace", profile.Namespace)
	return nil
}

func (r *CUDAExperimentReconciler) profileSummary(
	ctx context.Context,
	experiment *gpuv1alpha1.CUDAExperiment,
) (profileSummary, error) {
	configMap := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      profileSummaryConfigMapName(experiment),
		Namespace: experiment.Namespace,
	}, configMap)
	if apierrors.IsNotFound(err) {
		return profileSummary{}, errProfileSummaryMissing
	}
	if err != nil {
		return profileSummary{}, err
	}

	raw, ok := configMap.Data[profileSummaryConfigMapKey]
	if !ok || raw == "" {
		return profileSummary{}, errProfileSummaryMissing
	}
	return parseProfileSummary(raw)
}

func setWorkloadProfileSpec(profile *gpuv1alpha1.WorkloadProfile, experiment *gpuv1alpha1.CUDAExperiment) {
	profile.Spec.WorkloadImage = experiment.Spec.Image
	profile.Spec.Command = append([]string(nil), experiment.Spec.Command...)
	profile.Spec.Arguments = append([]string(nil), experiment.Spec.Arguments...)
	profile.Spec.SourceCUDAExperiment = experiment.Name
}

type profileSummary struct {
	BoundType gpuv1alpha1.WorkloadBoundType
	Metrics   map[string]string
}

func parseProfileSummary(raw string) (profileSummary, error) {
	var summary struct {
		BoundType string         `json:"boundType"`
		Metrics   map[string]any `json:"metrics"`
	}
	if err := json.Unmarshal([]byte(raw), &summary); err != nil {
		return profileSummary{}, errInvalidProfileSummary
	}

	boundType, ok := workloadBoundType(summary.BoundType)
	if !ok {
		return profileSummary{}, errInvalidProfileSummary
	}

	metrics := make(map[string]string, len(summary.Metrics))
	for key, value := range summary.Metrics {
		switch typed := value.(type) {
		case string:
			metrics[key] = typed
		case float64:
			metrics[key] = fmt.Sprintf("%g", typed)
		default:
			return profileSummary{}, errInvalidProfileSummary
		}
	}

	return profileSummary{
		BoundType: boundType,
		Metrics:   metrics,
	}, nil
}

func workloadBoundType(value string) (gpuv1alpha1.WorkloadBoundType, bool) {
	switch gpuv1alpha1.WorkloadBoundType(value) {
	case gpuv1alpha1.WorkloadBoundCompute,
		gpuv1alpha1.WorkloadBoundMemory,
		gpuv1alpha1.WorkloadBoundMixed,
		gpuv1alpha1.WorkloadBoundUnknown:
		return gpuv1alpha1.WorkloadBoundType(value), true
	default:
		return "", false
	}
}

func (r *CUDAExperimentReconciler) setExecutionJobCreatedCondition(ctx context.Context, experiment *gpuv1alpha1.CUDAExperiment) error {
	jobName := executionJobName(experiment)
	return r.updateCUDAExperimentStatus(ctx, experiment, jobName, metav1.Condition{
		Type:               executionJobConditionType,
		Status:             metav1.ConditionTrue,
		Reason:             "JobCreated",
		Message:            fmt.Sprintf("Execution Job %q exists", jobName),
		ObservedGeneration: experiment.Generation,
	})
}

func (r *CUDAExperimentReconciler) setProfilingPendingCondition(ctx context.Context, experiment *gpuv1alpha1.CUDAExperiment) error {
	return r.updateCUDAExperimentStatus(ctx, experiment, "", metav1.Condition{
		Type:               profilingPendingConditionType,
		Status:             metav1.ConditionTrue,
		Reason:             "WaitingForProfile",
		Message:            fmt.Sprintf("Profiling Job %q exists and is waiting for WorkloadProfile %q", profilingJobName(experiment), workloadProfileName(experiment)),
		ObservedGeneration: experiment.Generation,
	})
}

func (r *CUDAExperimentReconciler) setProfilingRunningCondition(ctx context.Context, experiment *gpuv1alpha1.CUDAExperiment) error {
	return r.updateCUDAExperimentStatus(ctx, experiment, "", metav1.Condition{
		Type:               profilingRunningConditionType,
		Status:             metav1.ConditionTrue,
		Reason:             "ProfilingJobRunning",
		Message:            fmt.Sprintf("Profiling Job %q is still running", profilingJobName(experiment)),
		ObservedGeneration: experiment.Generation,
	})
}

func (r *CUDAExperimentReconciler) setProfilingCompletedCondition(ctx context.Context, experiment *gpuv1alpha1.CUDAExperiment) error {
	return r.updateCUDAExperimentStatus(ctx, experiment, "", metav1.Condition{
		Type:               profilingCompletedConditionType,
		Status:             metav1.ConditionTrue,
		Reason:             "ProfilingJobCompleted",
		Message:            fmt.Sprintf("Profiling Job %q completed", profilingJobName(experiment)),
		ObservedGeneration: experiment.Generation,
	})
}

func (r *CUDAExperimentReconciler) setFailedCondition(
	ctx context.Context,
	experiment *gpuv1alpha1.CUDAExperiment,
	reason string,
	message string,
) error {
	return r.updateCUDAExperimentStatus(ctx, experiment, "", metav1.Condition{
		Type:               failedConditionType,
		Status:             metav1.ConditionTrue,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: experiment.Generation,
	})
}

func (r *CUDAExperimentReconciler) updateCUDAExperimentStatus(
	ctx context.Context,
	experiment *gpuv1alpha1.CUDAExperiment,
	executionJob string,
	condition metav1.Condition,
) error {
	current := &gpuv1alpha1.CUDAExperiment{}
	if err := r.Get(ctx, types.NamespacedName{Name: experiment.Name, Namespace: experiment.Namespace}, current); err != nil {
		return client.IgnoreNotFound(err)
	}

	existing := meta.FindStatusCondition(current.Status.Conditions, condition.Type)
	if existing != nil &&
		existing.Status == condition.Status &&
		existing.Reason == condition.Reason &&
		existing.ObservedGeneration == current.Generation &&
		(executionJob == "" || current.Status.ExecutionJobName == executionJob) {
		return nil
	}

	if executionJob != "" {
		current.Status.ExecutionJobName = executionJob
	}
	condition.ObservedGeneration = current.Generation
	meta.SetStatusCondition(&current.Status.Conditions, condition)

	if err := r.Status().Update(ctx, current); err != nil {
		return client.IgnoreNotFound(err)
	}
	log := ctrl.LoggerFrom(ctx)
	log.Info("Updated CUDAExperiment status", "condition", condition.Type)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CUDAExperimentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gpuv1alpha1.CUDAExperiment{}).
		Owns(&batchv1.Job{}).
		Named("cudaexperiment").
		Complete(r)
}

func (r *CUDAExperimentReconciler) executionJobForExperiment(experiment *gpuv1alpha1.CUDAExperiment) batchv1.Job {
	parallelism := max(experiment.Spec.Replicas, 1)
	gpuCount := gpuCountForExperiment(experiment)
	labels := labelsForExperiment(experiment)

	return batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      executionJobName(experiment),
			Namespace: experiment.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			Parallelism:  &parallelism,
			Completions:  &parallelism,
			BackoffLimit: ptrTo[int32](0),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					RuntimeClassName: ptrTo(runtimeClassNameForExperiment(experiment)),
					RestartPolicy:    corev1.RestartPolicyNever,
					Containers:       []corev1.Container{executionContainerForExperiment(experiment, gpuCount)},
				},
			},
		},
	}
}

func (r *CUDAExperimentReconciler) profilingJobForExperiment(experiment *gpuv1alpha1.CUDAExperiment) batchv1.Job {
	parallelism := max(experiment.Spec.Replicas, 1)
	gpuCount := gpuCountForExperiment(experiment)
	labels := labelsForExperiment(experiment)

	return batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      profilingJobName(experiment),
			Namespace: experiment.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			Parallelism:  &parallelism,
			Completions:  &parallelism,
			BackoffLimit: ptrTo[int32](0),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					RuntimeClassName: ptrTo(runtimeClassNameForExperiment(experiment)),
					RestartPolicy:    corev1.RestartPolicyNever,
					InitContainers:   []corev1.Container{workloadStagingInitContainerForExperiment(experiment)},
					Containers:       []corev1.Container{profilingRunnerForExperiment(experiment, gpuCount, r.nsightComputeImage())},
					Volumes:          []corev1.Volume{sharedWorkloadVolume()},
				},
			},
		},
	}
}

func (r *CUDAExperimentReconciler) nsightComputeImage() string {
	if r.NsightComputeImage != "" {
		return r.NsightComputeImage
	}
	return defaultNsightComputeImage
}

func executionContainerForExperiment(experiment *gpuv1alpha1.CUDAExperiment, gpuCount int32) corev1.Container {
	return corev1.Container{
		Name:    "execution",
		Image:   experiment.Spec.Image,
		Command: experiment.Spec.Command,
		Args:    experiment.Spec.Arguments,
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceName("nvidia.com/gpu"): *resource.NewQuantity(int64(gpuCount), resource.DecimalSI),
			},
		},
	}
}

func workloadStagingInitContainerForExperiment(experiment *gpuv1alpha1.CUDAExperiment) corev1.Container {
	return corev1.Container{
		Name:    "stage-workload",
		Image:   experiment.Spec.Image,
		Command: []string{"/bin/sh", "-c", workloadStagingScript(workloadExecutablePath(experiment))},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      sharedWorkloadVolumeName,
				MountPath: sharedWorkloadMountPath,
			},
		},
	}
}

func profilingRunnerForExperiment(experiment *gpuv1alpha1.CUDAExperiment, gpuCount int32, nsightComputeImage string) corev1.Container {
	return corev1.Container{
		Name:            "profiling-runner",
		Image:           nsightComputeImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"/scripts/profile.sh", sharedWorkloadMountPath + "/workload", profilingReportPath(experiment)},
		Args:            experiment.Spec.Arguments,
		Env: []corev1.EnvVar{
			{
				Name:  "KRATOS_EXPERIMENT_NAME",
				Value: experiment.Name,
			},
			{
				Name:  "KRATOS_EXPERIMENT_NAMESPACE",
				Value: experiment.Namespace,
			},
			{
				Name:  "KRATOS_PROFILE_SUMMARY_CONFIGMAP",
				Value: profileSummaryConfigMapName(experiment),
			},
			{
				Name:  "KRATOS_PROFILE_SUMMARY_KEY",
				Value: profileSummaryConfigMapKey,
			},
		},
		SecurityContext: &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{"SYS_ADMIN"},
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      sharedWorkloadVolumeName,
				MountPath: sharedWorkloadMountPath,
			},
		},
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceName("nvidia.com/gpu"): *resource.NewQuantity(int64(gpuCount), resource.DecimalSI),
			},
		},
	}
}

func gpuCountForExperiment(experiment *gpuv1alpha1.CUDAExperiment) int32 {
	gpuCount := experiment.Spec.NumberOfGPUs
	if gpuCount < 1 {
		gpuCount = experiment.Spec.GPURequired
	}
	if gpuCount < 1 {
		return 1
	}
	return gpuCount
}

func labelsForExperiment(experiment *gpuv1alpha1.CUDAExperiment) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "kratos",
		"app.kubernetes.io/managed-by": "kratos-controller",
		"gpu.scheduler.io/experiment":  experiment.Name,
	}
}

func sharedWorkloadVolume() corev1.Volume {
	return corev1.Volume{
		Name: sharedWorkloadVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
}

func isJobComplete(job *batchv1.Job) bool {
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobComplete && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func isJobFailed(job *batchv1.Job) bool {
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func workloadExecutablePath(experiment *gpuv1alpha1.CUDAExperiment) string {
	if len(experiment.Spec.Command) > 0 && experiment.Spec.Command[0] != "" {
		return experiment.Spec.Command[0]
	}
	return defaultWorkloadExecutable
}

func workloadStagingScript(workloadPath string) string {
	return fmt.Sprintf(`set -eu
if [ ! -x %[1]s ]; then
  echo "Workload executable %[1]s is not executable; set spec.command[0] to the CUDA executable path for profiling" >&2
  exit 1
fi
cp %[1]s %[2]s/workload
chmod +x %[2]s/workload
echo "Staged workload executable for Nsight Compute profiling runner"
`, shellQuote(workloadPath), sharedWorkloadMountPath)
}

func shellQuote(value string) string {
	var quoted strings.Builder
	quoted.WriteByte('\'')
	for _, r := range value {
		if r == '\'' {
			quoted.WriteString("'\\''")
			continue
		}
		quoted.WriteRune(r)
	}
	quoted.WriteByte('\'')
	return quoted.String()
}

func profilingReportPath(experiment *gpuv1alpha1.CUDAExperiment) string {
	return sharedWorkloadMountPath + "/nsight-compute/" + experiment.Name
}

func runtimeClassNameForExperiment(experiment *gpuv1alpha1.CUDAExperiment) string {
	if experiment.Spec.RuntimeClassName != "" {
		return experiment.Spec.RuntimeClassName
	}
	return defaultRuntimeClassName
}

func ptrTo[T any](value T) *T {
	return &value
}
