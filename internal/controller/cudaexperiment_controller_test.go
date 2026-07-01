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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gpuv1alpha1 "github.com/chirichexe/kratos/api/v1alpha1"
)

const (
	jobCompleteReason           = "Completed"
	jobCompletionsReachedReason = "CompletionsReached"
	resourceNamespace           = "default"
)

var _ = Describe("CUDAExperiment Controller", func() {
	ctx := context.Background()

	It("creates only an ExecutionJob when profiling is disabled and remains idempotent", func() {
		experiment := createCUDAExperiment(ctx, "test-resource-no-profile", false)
		reconciler := cudaExperimentReconciler()

		reconcileExperiment(ctx, reconciler, experiment)

		expectJobExists(ctx, executionJobName(experiment), experiment.Name)
		expectJobNotFound(ctx, profilingJobName(experiment))
		expectOneJobForExperiment(ctx, experiment.Name)
		expectCondition(ctx, experiment, executionJobConditionType, "JobCreated")
		expectOneCondition(ctx, experiment, executionJobConditionType)
	})

	It("creates only a ProfilingJob when profiling is enabled and the WorkloadProfile is missing", func() {
		experiment := createCUDAExperiment(ctx, "test-resource-profile-missing", true)
		reconciler := cudaExperimentReconciler()

		reconcileExperiment(ctx, reconciler, experiment)

		expectJobExists(ctx, profilingJobName(experiment), experiment.Name)
		expectJobNotFound(ctx, executionJobName(experiment))
		expectOneJobForExperiment(ctx, experiment.Name)
		expectCondition(ctx, experiment, profilingPendingConditionType, "WaitingForProfile")
		expectCondition(ctx, experiment, profilingRunningConditionType, "ProfilingJobRunning")
		expectOneCondition(ctx, experiment, profilingPendingConditionType)
		expectOneCondition(ctx, experiment, profilingRunningConditionType)
	})

	It("creates only an ExecutionJob when profiling is enabled and the WorkloadProfile exists", func() {
		experiment := createCUDAExperiment(ctx, "test-resource-profile-ready", true)
		createWorkloadProfile(ctx, experiment)
		reconciler := cudaExperimentReconciler()

		reconcileExperiment(ctx, reconciler, experiment)

		expectJobExists(ctx, executionJobName(experiment), experiment.Name)
		expectJobNotFound(ctx, profilingJobName(experiment))
		expectOneJobForExperiment(ctx, experiment.Name)
		expectCondition(ctx, experiment, executionJobConditionType, "JobCreated")
		expectOneCondition(ctx, experiment, executionJobConditionType)
	})

	It("does not create an ExecutionJob while an existing ProfilingJob is still running", func() {
		experiment := createCUDAExperiment(ctx, "test-resource-profile-running", true)
		reconciler := cudaExperimentReconciler()
		createProfilingJob(ctx, reconciler, experiment, nil)

		reconcileExperiment(ctx, reconciler, experiment)

		expectJobExists(ctx, profilingJobName(experiment), experiment.Name)
		expectJobNotFound(ctx, executionJobName(experiment))
		expectOneJobForExperiment(ctx, experiment.Name)
		expectCondition(ctx, experiment, profilingRunningConditionType, "ProfilingJobRunning")
		expectOneCondition(ctx, experiment, profilingRunningConditionType)
	})

	It("does not create an ExecutionJob when the ProfilingJob completed before profile generation exists", func() {
		experiment := createCUDAExperiment(ctx, "test-resource-profile-summary-missing", true)
		reconciler := cudaExperimentReconciler()
		createProfilingJob(ctx, reconciler, experiment, []batchv1.JobCondition{
			{
				Type:   batchv1.JobSuccessCriteriaMet,
				Status: corev1.ConditionTrue,
				Reason: jobCompletionsReachedReason,
			},
			{
				Type:   batchv1.JobComplete,
				Status: corev1.ConditionTrue,
				Reason: jobCompleteReason,
			},
		})

		reconcileExperiment(ctx, reconciler, experiment)

		expectJobExists(ctx, profilingJobName(experiment), experiment.Name)
		expectJobNotFound(ctx, executionJobName(experiment))
		expectOneJobForExperiment(ctx, experiment.Name)
		expectWorkloadProfileNotFound(ctx, experiment)
		expectCondition(ctx, experiment, failedConditionType, "ProfileSummaryMissing")
		expectOneCondition(ctx, experiment, failedConditionType)
	})

	It("creates a WorkloadProfile from a completed ProfilingJob summary before starting execution", func() {
		experiment := createCUDAExperiment(ctx, "test-resource-profile-summary-valid", true)
		reconciler := cudaExperimentReconciler()
		createProfilingJob(ctx, reconciler, experiment, []batchv1.JobCondition{
			{
				Type:   batchv1.JobSuccessCriteriaMet,
				Status: corev1.ConditionTrue,
				Reason: jobCompletionsReachedReason,
			},
			{
				Type:   batchv1.JobComplete,
				Status: corev1.ConditionTrue,
				Reason: jobCompleteReason,
			},
		})
		createProfileSummaryConfigMap(ctx, experiment, `{
			"boundType": "memory-bound",
			"metrics": {
				"smThroughput": "35.2",
				"dramThroughput": "91.4",
				"achievedOccupancy": "0.55"
			}
		}`)

		reconcileExperimentOnce(ctx, reconciler, experiment)

		expectJobExists(ctx, profilingJobName(experiment), experiment.Name)
		expectJobNotFound(ctx, executionJobName(experiment))
		expectOneJobForExperiment(ctx, experiment.Name)
		expectWorkloadProfile(ctx, experiment, gpuv1alpha1.WorkloadBoundMemory, map[string]string{
			"smThroughput":      "35.2",
			"dramThroughput":    "91.4",
			"achievedOccupancy": "0.55",
		})
		expectOneWorkloadProfileForExperiment(ctx, experiment.Name)
		expectCondition(ctx, experiment, profilingCompletedConditionType, "ProfilingJobCompleted")
		expectOneCondition(ctx, experiment, profilingCompletedConditionType)

		reconcileExperiment(ctx, reconciler, experiment)

		expectOneWorkloadProfileForExperiment(ctx, experiment.Name)
		expectJobExists(ctx, executionJobName(experiment), experiment.Name)
		expectCondition(ctx, experiment, executionJobConditionType, "JobCreated")
	})

	It("does not create a WorkloadProfile or ExecutionJob from invalid profile summary JSON", func() {
		experiment := createCUDAExperiment(ctx, "test-resource-profile-summary-invalid", true)
		reconciler := cudaExperimentReconciler()
		createProfilingJob(ctx, reconciler, experiment, []batchv1.JobCondition{
			{
				Type:   batchv1.JobSuccessCriteriaMet,
				Status: corev1.ConditionTrue,
				Reason: jobCompletionsReachedReason,
			},
			{
				Type:   batchv1.JobComplete,
				Status: corev1.ConditionTrue,
				Reason: jobCompleteReason,
			},
		})
		createProfileSummaryConfigMap(ctx, experiment, `{"boundType": "io-bound", "metrics": {"smThroughput": "35.2"}}`)

		reconcileExperiment(ctx, reconciler, experiment)

		expectJobExists(ctx, profilingJobName(experiment), experiment.Name)
		expectJobNotFound(ctx, executionJobName(experiment))
		expectOneJobForExperiment(ctx, experiment.Name)
		expectWorkloadProfileNotFound(ctx, experiment)
		expectCondition(ctx, experiment, failedConditionType, "InvalidProfileSummary")
		expectOneCondition(ctx, experiment, failedConditionType)
	})

	It("does not create an ExecutionJob when the ProfilingJob failed", func() {
		experiment := createCUDAExperiment(ctx, "test-resource-profile-failed", true)
		reconciler := cudaExperimentReconciler()
		createProfilingJob(ctx, reconciler, experiment, []batchv1.JobCondition{
			{
				Type:   batchv1.JobFailureTarget,
				Status: corev1.ConditionTrue,
				Reason: "BackoffLimitExceeded",
			},
			{
				Type:   batchv1.JobFailed,
				Status: corev1.ConditionTrue,
				Reason: "BackoffLimitExceeded",
			},
		})

		reconcileExperiment(ctx, reconciler, experiment)

		expectJobExists(ctx, profilingJobName(experiment), experiment.Name)
		expectJobNotFound(ctx, executionJobName(experiment))
		expectOneJobForExperiment(ctx, experiment.Name)
		expectCondition(ctx, experiment, failedConditionType, "ProfilingJobFailed")
		expectOneCondition(ctx, experiment, failedConditionType)
	})
})

func cudaExperimentReconciler() *CUDAExperimentReconciler {
	return &CUDAExperimentReconciler{
		Client: k8sClient,
		Scheme: k8sClient.Scheme(),
	}
}

func createCUDAExperiment(ctx context.Context, name string, profilingEnabled bool) *gpuv1alpha1.CUDAExperiment {
	experiment := &gpuv1alpha1.CUDAExperiment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: resourceNamespace,
		},
		Spec: gpuv1alpha1.CUDAExperimentSpec{
			Image:            testCUDAImage,
			Command:          []string{defaultWorkloadExecutable},
			Arguments:        []string{"--iterations=1"},
			Replicas:         1,
			GPURequired:      1,
			RuntimeClassName: defaultRuntimeClassName,
			NumberOfGPUs:     1,
			ProfilingEnabled: profilingEnabled,
		},
	}

	Expect(k8sClient.Create(ctx, experiment)).To(Succeed())
	DeferCleanup(func() {
		err := k8sClient.Delete(ctx, experiment)
		Expect(client.IgnoreNotFound(err)).To(Succeed())
	})

	return experiment
}

func createWorkloadProfile(ctx context.Context, experiment *gpuv1alpha1.CUDAExperiment) {
	profile := &gpuv1alpha1.WorkloadProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workloadProfileName(experiment),
			Namespace: experiment.Namespace,
		},
		Spec: gpuv1alpha1.WorkloadProfileSpec{
			WorkloadImage:        experiment.Spec.Image,
			Command:              experiment.Spec.Command,
			Arguments:            experiment.Spec.Arguments,
			SourceCUDAExperiment: experiment.Name,
		},
	}

	Expect(k8sClient.Create(ctx, profile)).To(Succeed())
	DeferCleanup(func() {
		err := k8sClient.Delete(ctx, profile)
		Expect(client.IgnoreNotFound(err)).To(Succeed())
	})
}

func createProfilingJob(
	ctx context.Context,
	reconciler *CUDAExperimentReconciler,
	experiment *gpuv1alpha1.CUDAExperiment,
	conditions []batchv1.JobCondition,
) {
	job := reconciler.profilingJobForExperiment(experiment)
	Expect(ctrl.SetControllerReference(experiment, &job, reconciler.Scheme)).To(Succeed())
	Expect(k8sClient.Create(ctx, &job)).To(Succeed())
	DeferCleanup(func() {
		err := k8sClient.Delete(ctx, &job)
		Expect(client.IgnoreNotFound(err)).To(Succeed())
	})

	if len(conditions) == 0 {
		return
	}

	current := &batchv1.Job{}
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: job.Name, Namespace: job.Namespace}, current)).To(Succeed())
	now := metav1.Now()
	current.Status.StartTime = &now
	if hasJobCondition(conditions, batchv1.JobComplete) {
		current.Status.CompletionTime = &now
	}
	for i := range conditions {
		conditions[i].LastTransitionTime = now
	}
	current.Status.Conditions = conditions
	Expect(k8sClient.Status().Update(ctx, current)).To(Succeed())
}

func createProfileSummaryConfigMap(ctx context.Context, experiment *gpuv1alpha1.CUDAExperiment, summary string) {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      profileSummaryConfigMapName(experiment),
			Namespace: experiment.Namespace,
		},
		Data: map[string]string{
			profileSummaryConfigMapKey: summary,
		},
	}

	Expect(k8sClient.Create(ctx, configMap)).To(Succeed())
	DeferCleanup(func() {
		err := k8sClient.Delete(ctx, configMap)
		Expect(client.IgnoreNotFound(err)).To(Succeed())
	})
}

func hasJobCondition(conditions []batchv1.JobCondition, conditionType batchv1.JobConditionType) bool {
	for _, condition := range conditions {
		if condition.Type == conditionType && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func reconcileExperimentOnce(
	ctx context.Context,
	reconciler *CUDAExperimentReconciler,
	experiment *gpuv1alpha1.CUDAExperiment,
) {
	request := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      experiment.Name,
			Namespace: experiment.Namespace,
		},
	}
	_, err := reconciler.Reconcile(ctx, request)
	Expect(err).NotTo(HaveOccurred())
}

func reconcileExperiment(
	ctx context.Context,
	reconciler *CUDAExperimentReconciler,
	experiment *gpuv1alpha1.CUDAExperiment,
) {
	for range 3 {
		reconcileExperimentOnce(ctx, reconciler, experiment)
	}
}

func expectJobExists(ctx context.Context, name, experimentName string) {
	job := &batchv1.Job{}
	err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: resourceNamespace}, job)
	Expect(err).NotTo(HaveOccurred())
	Expect(job.OwnerReferences).To(HaveLen(1))
	Expect(job.OwnerReferences[0].Name).To(Equal(experimentName))
}

func expectJobNotFound(ctx context.Context, name string) {
	job := &batchv1.Job{}
	err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: resourceNamespace}, job)
	Expect(errors.IsNotFound(err)).To(BeTrue())
}

func expectOneJobForExperiment(ctx context.Context, experimentName string) {
	jobs := &batchv1.JobList{}
	Expect(k8sClient.List(
		ctx,
		jobs,
		client.InNamespace(resourceNamespace),
		client.MatchingLabels{"gpu.scheduler.io/experiment": experimentName},
	)).To(Succeed())
	Expect(jobs.Items).To(HaveLen(1))
}

func expectWorkloadProfile(
	ctx context.Context,
	experiment *gpuv1alpha1.CUDAExperiment,
	boundType gpuv1alpha1.WorkloadBoundType,
	metrics map[string]string,
) {
	profile := &gpuv1alpha1.WorkloadProfile{}
	Expect(k8sClient.Get(ctx, types.NamespacedName{
		Name:      workloadProfileName(experiment),
		Namespace: experiment.Namespace,
	}, profile)).To(Succeed())
	Expect(profile.OwnerReferences).To(HaveLen(1))
	Expect(profile.OwnerReferences[0].Name).To(Equal(experiment.Name))
	Expect(profile.Spec.WorkloadImage).To(Equal(experiment.Spec.Image))
	Expect(profile.Spec.Command).To(Equal(experiment.Spec.Command))
	Expect(profile.Spec.Arguments).To(Equal(experiment.Spec.Arguments))
	Expect(profile.Spec.SourceCUDAExperiment).To(Equal(experiment.Name))
	Expect(profile.Status.BoundType).To(Equal(boundType))
	Expect(profile.Status.Metrics).To(Equal(metrics))
}

func expectWorkloadProfileNotFound(ctx context.Context, experiment *gpuv1alpha1.CUDAExperiment) {
	profile := &gpuv1alpha1.WorkloadProfile{}
	err := k8sClient.Get(ctx, types.NamespacedName{
		Name:      workloadProfileName(experiment),
		Namespace: experiment.Namespace,
	}, profile)
	Expect(errors.IsNotFound(err)).To(BeTrue())
}

func expectOneWorkloadProfileForExperiment(ctx context.Context, experimentName string) {
	profiles := &gpuv1alpha1.WorkloadProfileList{}
	Expect(k8sClient.List(
		ctx,
		profiles,
		client.InNamespace(resourceNamespace),
	)).To(Succeed())

	count := 0
	for _, profile := range profiles.Items {
		if profile.Spec.SourceCUDAExperiment == experimentName {
			count++
		}
	}
	Expect(count).To(Equal(1))
}

func expectCondition(ctx context.Context, experiment *gpuv1alpha1.CUDAExperiment, conditionType, reason string) {
	current := &gpuv1alpha1.CUDAExperiment{}
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: experiment.Name, Namespace: experiment.Namespace}, current)).To(Succeed())

	condition := meta.FindStatusCondition(current.Status.Conditions, conditionType)
	Expect(condition).NotTo(BeNil())
	Expect(condition.Status).To(Equal(metav1.ConditionTrue))
	Expect(condition.Reason).To(Equal(reason))
}

func expectOneCondition(ctx context.Context, experiment *gpuv1alpha1.CUDAExperiment, conditionType string) {
	current := &gpuv1alpha1.CUDAExperiment{}
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: experiment.Name, Namespace: experiment.Namespace}, current)).To(Succeed())

	count := 0
	for _, condition := range current.Status.Conditions {
		if condition.Type == conditionType {
			count++
		}
	}
	Expect(count).To(Equal(1))
}
