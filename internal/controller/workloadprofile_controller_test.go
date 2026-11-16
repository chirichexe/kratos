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
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gpuv1alpha1 "github.com/chirichexe/kratos/api/v1alpha1"
)

var _ = Describe("WorkloadProfile Controller", func() {
	ctx := context.Background()

	It("summarizes execution feedback from the Jobs that reused this profile", func() {
		profile := createBareWorkloadProfile(ctx, "test-profile-feedback")
		older := metav1.NewTime(time.Now().Add(-time.Hour))
		newer := metav1.NewTime(time.Now())
		createLabeledExecutionJob(ctx, "feedback-job-older", profile.Name, []batchv1.JobCondition{
			{Type: batchv1.JobComplete, Status: corev1.ConditionTrue, LastTransitionTime: older},
		})
		createLabeledExecutionJob(ctx, "feedback-job-newer", profile.Name, []batchv1.JobCondition{
			{Type: batchv1.JobFailed, Status: corev1.ConditionTrue, LastTransitionTime: newer},
		})
		reconciler := workloadProfileReconciler()

		reconcileWorkloadProfileOnce(ctx, reconciler, profile)

		updated := &gpuv1alpha1.WorkloadProfile{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: profile.Name, Namespace: resourceNamespace}, updated)).To(Succeed())
		Expect(updated.Status.ExecutionCount).To(Equal(int32(2)))
		Expect(updated.Status.LastExecutionOutcome).To(Equal(gpuv1alpha1.ExecutionFailed))
		Expect(updated.Status.LastExecutionTime.Time).To(BeTemporally("~", newer.Time, time.Second))
	})

	It("ignores Jobs that have not finished yet", func() {
		profile := createBareWorkloadProfile(ctx, "test-profile-running")
		createLabeledExecutionJob(ctx, "running-job", profile.Name, nil)
		reconciler := workloadProfileReconciler()

		reconcileWorkloadProfileOnce(ctx, reconciler, profile)

		updated := &gpuv1alpha1.WorkloadProfile{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: profile.Name, Namespace: resourceNamespace}, updated)).To(Succeed())
		Expect(updated.Status.ExecutionCount).To(Equal(int32(0)))
		Expect(updated.Status.LastExecutionTime).To(BeNil())
	})

	It("ignores Jobs labeled for a different profile", func() {
		profile := createBareWorkloadProfile(ctx, "test-profile-isolated")
		createLabeledExecutionJob(ctx, "other-profile-job", "some-other-profile", []batchv1.JobCondition{
			{Type: batchv1.JobComplete, Status: corev1.ConditionTrue, LastTransitionTime: metav1.Now()},
		})
		reconciler := workloadProfileReconciler()

		reconcileWorkloadProfileOnce(ctx, reconciler, profile)

		updated := &gpuv1alpha1.WorkloadProfile{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: profile.Name, Namespace: resourceNamespace}, updated)).To(Succeed())
		Expect(updated.Status.ExecutionCount).To(Equal(int32(0)))
	})
})

func workloadProfileReconciler() *WorkloadProfileReconciler {
	return &WorkloadProfileReconciler{
		Client: k8sClient,
		Scheme: k8sClient.Scheme(),
	}
}

func reconcileWorkloadProfileOnce(ctx context.Context, reconciler *WorkloadProfileReconciler, profile *gpuv1alpha1.WorkloadProfile) {
	_, err := reconciler.Reconcile(ctx, reconcile.Request{
		NamespacedName: types.NamespacedName{Name: profile.Name, Namespace: profile.Namespace},
	})
	Expect(err).NotTo(HaveOccurred())
}

func createBareWorkloadProfile(ctx context.Context, name string) *gpuv1alpha1.WorkloadProfile {
	profile := &gpuv1alpha1.WorkloadProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: resourceNamespace,
		},
		Spec: gpuv1alpha1.WorkloadProfileSpec{
			WorkloadImage: testCUDAImage,
		},
	}
	Expect(k8sClient.Create(ctx, profile)).To(Succeed())
	DeferCleanup(func() {
		err := k8sClient.Delete(ctx, profile)
		Expect(client.IgnoreNotFound(err)).To(Succeed())
	})
	return profile
}

func createLabeledExecutionJob(ctx context.Context, name, profileName string, conditions []batchv1.JobCondition) {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: resourceNamespace,
			Labels:    map[string]string{workloadProfileLabel: profileName},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{Name: "c", Image: testCUDAImage},
					},
				},
			},
		},
	}
	Expect(k8sClient.Create(ctx, job)).To(Succeed())
	DeferCleanup(func() {
		err := k8sClient.Delete(ctx, job)
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
		conditions = append(conditions, batchv1.JobCondition{
			Type:               batchv1.JobSuccessCriteriaMet,
			Status:             corev1.ConditionTrue,
			LastTransitionTime: now,
		})
	}
	if hasJobCondition(conditions, batchv1.JobFailed) {
		conditions = append(conditions, batchv1.JobCondition{
			Type:               batchv1.JobFailureTarget,
			Status:             corev1.ConditionTrue,
			LastTransitionTime: now,
		})
	}
	current.Status.Conditions = conditions
	Expect(k8sClient.Status().Update(ctx, current)).To(Succeed())
}

// ---------------------------------------------------------------------------
// pure helper tests (no cluster needed)
// ---------------------------------------------------------------------------

func TestSummarizeExecutionFeedback(t *testing.T) {
	older := metav1.NewTime(time.Now().Add(-time.Hour))
	newer := metav1.NewTime(time.Now())

	jobs := []batchv1.Job{
		{Status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{
			{Type: batchv1.JobComplete, Status: corev1.ConditionTrue, LastTransitionTime: older},
		}}},
		{Status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{
			{Type: batchv1.JobFailed, Status: corev1.ConditionTrue, LastTransitionTime: newer},
		}}},
		{Status: batchv1.JobStatus{}}, // still running, no terminal condition
	}

	count, lastTime, lastOutcome := summarizeExecutionFeedback(jobs)

	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
	if lastTime == nil || !lastTime.Time.Equal(newer.Time) {
		t.Fatalf("lastTime = %v, want %v", lastTime, newer.Time)
	}
	if lastOutcome != gpuv1alpha1.ExecutionFailed {
		t.Fatalf("lastOutcome = %q, want %q", lastOutcome, gpuv1alpha1.ExecutionFailed)
	}
}

func TestSummarizeExecutionFeedbackNoJobs(t *testing.T) {
	count, lastTime, lastOutcome := summarizeExecutionFeedback(nil)
	if count != 0 || lastTime != nil || lastOutcome != "" {
		t.Fatalf("got (%d, %v, %q), want (0, nil, \"\")", count, lastTime, lastOutcome)
	}
}
