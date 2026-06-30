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
	executionJobConditionType = "ExecutionJobCreated"
	executionJobNameSuffix    = "-execution"
	defaultRuntimeClassName   = "nvidia"
	defaultNsightComputeImage = "kratos-nsight-compute-poc:latest"
	defaultWorkloadExecutable = "/cuda-samples/vectorAdd"
	sharedWorkloadVolumeName  = "workload"
	sharedWorkloadMountPath   = "/shared"
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
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.24.1/pkg/reconcile
func (r *CUDAExperimentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	var experiment gpuv1alpha1.CUDAExperiment
	if err := r.Get(ctx, req.NamespacedName, &experiment); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	jobName := executionJobName(&experiment)
	var job batchv1.Job
	err := r.Get(ctx, types.NamespacedName{Name: jobName, Namespace: experiment.Namespace}, &job)
	if apierrors.IsNotFound(err) {
		job = r.executionJobForExperiment(&experiment)
		if err := ctrl.SetControllerReference(&experiment, &job, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.Create(ctx, &job); err != nil {
			return ctrl.Result{}, err
		}
		log.Info("Created Job", "name", job.Name, "namespace", job.Namespace)
	} else if err != nil {
		return ctrl.Result{}, err
	}

	condition := meta.FindStatusCondition(experiment.Status.Conditions, executionJobConditionType)
	if condition != nil &&
		condition.ObservedGeneration == experiment.Generation &&
		experiment.Status.ExecutionJobName == jobName {
		return ctrl.Result{}, nil
	}

	experiment.Status.ExecutionJobName = jobName
	meta.SetStatusCondition(&experiment.Status.Conditions, metav1.Condition{
		Type:               executionJobConditionType,
		Status:             metav1.ConditionTrue,
		Reason:             "JobCreated",
		Message:            fmt.Sprintf("Execution Job %q exists.", jobName),
		ObservedGeneration: experiment.Generation,
	})

	if err := r.Status().Update(ctx, &experiment); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.Info("Updated CUDAExperiment status", "condition", executionJobConditionType, "job", jobName)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CUDAExperimentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gpuv1alpha1.CUDAExperiment{}).
		Owns(&batchv1.Job{}).
		Named("cudaexperiment").
		Complete(r)
}

func executionJobName(experiment *gpuv1alpha1.CUDAExperiment) string {
	return experiment.Name + executionJobNameSuffix
}

func (r *CUDAExperimentReconciler) executionJobForExperiment(experiment *gpuv1alpha1.CUDAExperiment) batchv1.Job {
	parallelism := max(experiment.Spec.Replicas, 1)

	gpuCount := experiment.Spec.NumberOfGPUs
	if gpuCount < 1 {
		gpuCount = experiment.Spec.GPURequired
	}
	if gpuCount < 1 {
		gpuCount = 1
	}

	labels := map[string]string{
		"app.kubernetes.io/name":       "kratos",
		"app.kubernetes.io/managed-by": "kratos-controller",
		"gpu.scheduler.io/experiment":  experiment.Name,
	}

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
					InitContainers:   initContainersForExperiment(experiment),
					Containers:       containersForExperiment(experiment, gpuCount, r.nsightComputeImage()),
					Volumes:          volumesForExperiment(experiment),
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

func containersForExperiment(experiment *gpuv1alpha1.CUDAExperiment, gpuCount int32, nsightComputeImage string) []corev1.Container {
	if !experiment.Spec.ProfilingEnabled {
		return []corev1.Container{executionContainerForExperiment(experiment, gpuCount)}
	}
	return []corev1.Container{
		profilingRunnerForExperiment(experiment, gpuCount, nsightComputeImage),
	}
}

func initContainersForExperiment(experiment *gpuv1alpha1.CUDAExperiment) []corev1.Container {
	if !experiment.Spec.ProfilingEnabled {
		return nil
	}
	return []corev1.Container{workloadStagingInitContainerForExperiment(experiment)}
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

func volumesForExperiment(experiment *gpuv1alpha1.CUDAExperiment) []corev1.Volume {
	if !experiment.Spec.ProfilingEnabled {
		return nil
	}
	return []corev1.Volume{
		{
			Name: sharedWorkloadVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}
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
