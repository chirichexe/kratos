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
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	gpuv1alpha1 "github.com/chirichexe/kratos/api/v1alpha1"
)

// WorkloadProfileReconciler reconciles a WorkloadProfile object. It closes
// the feedback loop the one-shot profiling Job cannot: every execution Job
// that reused this profile for node placement (labeled with
// workloadProfileLabel) is watched, and the profile's status is kept
// summarizing how often it has been reused and how those runs fared.
type WorkloadProfileReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=gpu.scheduler.io,resources=workloadprofiles,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=gpu.scheduler.io,resources=workloadprofiles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch

// Reconcile recomputes the execution feedback summary (executionCount,
// lastExecutionTime, lastExecutionOutcome) for a WorkloadProfile from the
// finished execution Jobs that reused it, and writes it back only when it
// changed.
func (r *WorkloadProfileReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var profile gpuv1alpha1.WorkloadProfile
	if err := r.Get(ctx, req.NamespacedName, &profile); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	var jobs batchv1.JobList
	if err := r.List(ctx, &jobs,
		client.InNamespace(profile.Namespace),
		client.MatchingLabels{workloadProfileLabel: profile.Name},
	); err != nil {
		return ctrl.Result{}, err
	}

	count, lastTime, lastOutcome := summarizeExecutionFeedback(jobs.Items)

	if profile.Status.ExecutionCount == int32(count) &&
		timesEqual(profile.Status.LastExecutionTime, lastTime) &&
		profile.Status.LastExecutionOutcome == lastOutcome {
		return ctrl.Result{}, nil
	}

	profile.Status.ExecutionCount = int32(count)
	profile.Status.LastExecutionTime = lastTime
	profile.Status.LastExecutionOutcome = lastOutcome
	if err := r.Status().Update(ctx, &profile); err != nil {
		return ctrl.Result{}, err
	}

	ctrl.LoggerFrom(ctx).Info("Updated WorkloadProfile execution feedback",
		"name", profile.Name, "executionCount", count, "lastExecutionOutcome", lastOutcome)
	return ctrl.Result{}, nil
}

// summarizeExecutionFeedback counts the finished (Complete or Failed) Jobs
// and reports the time and outcome of whichever finished most recently.
func summarizeExecutionFeedback(jobs []batchv1.Job) (count int, lastTime *metav1.Time, lastOutcome gpuv1alpha1.ExecutionOutcome) {
	for _, job := range jobs {
		finishedAt, outcome, finished := jobOutcome(job)
		if !finished {
			continue
		}
		count++
		if lastTime == nil || finishedAt.After(lastTime.Time) {
			t := finishedAt
			lastTime = &metav1.Time{Time: t}
			lastOutcome = outcome
		}
	}
	return count, lastTime, lastOutcome
}

// jobOutcome reports whether a Job has finished and, if so, when and with
// what outcome, based on its standard batch/v1 conditions.
func jobOutcome(job batchv1.Job) (finishedAt time.Time, outcome gpuv1alpha1.ExecutionOutcome, finished bool) {
	for _, condition := range job.Status.Conditions {
		switch {
		case condition.Type == batchv1.JobComplete && condition.Status == corev1.ConditionTrue:
			return condition.LastTransitionTime.Time, gpuv1alpha1.ExecutionSucceeded, true
		case condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue:
			return condition.LastTransitionTime.Time, gpuv1alpha1.ExecutionFailed, true
		}
	}
	return time.Time{}, "", false
}

// timesEqual reports whether two *metav1.Time pointers represent the same
// instant, treating two nils as equal.
func timesEqual(a, b *metav1.Time) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Equal(b)
}

// jobToWorkloadProfileRequest maps an execution Job back to the
// WorkloadProfile it reused, via workloadProfileLabel, so changes to that
// Job (e.g. reaching Complete or Failed) trigger a reconcile of the profile.
func jobToWorkloadProfileRequest(_ context.Context, obj client.Object) []reconcile.Request {
	job, ok := obj.(*batchv1.Job)
	if !ok {
		return nil
	}
	profileName, ok := job.Labels[workloadProfileLabel]
	if !ok || profileName == "" {
		return nil
	}
	return []reconcile.Request{{
		NamespacedName: types.NamespacedName{Name: profileName, Namespace: job.Namespace},
	}}
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkloadProfileReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gpuv1alpha1.WorkloadProfile{}).
		Watches(&batchv1.Job{}, handler.EnqueueRequestsFromMapFunc(jobToWorkloadProfileRequest)).
		Named("workloadprofile").
		Complete(r)
}
