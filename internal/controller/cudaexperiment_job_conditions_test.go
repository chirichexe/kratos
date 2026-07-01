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
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestIsJobComplete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		job  *batchv1.Job
		want bool
	}{
		{
			name: "complete true",
			job: jobWithConditions(batchv1.JobCondition{
				Type:   batchv1.JobComplete,
				Status: corev1.ConditionTrue,
			}),
			want: true,
		},
		{
			name: "complete false",
			job: jobWithConditions(batchv1.JobCondition{
				Type:   batchv1.JobComplete,
				Status: corev1.ConditionFalse,
			}),
			want: false,
		},
		{
			name: "failed only",
			job: jobWithConditions(batchv1.JobCondition{
				Type:   batchv1.JobFailed,
				Status: corev1.ConditionTrue,
			}),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := isJobComplete(tt.job); got != tt.want {
				t.Fatalf("isJobComplete() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestIsJobFailed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		job  *batchv1.Job
		want bool
	}{
		{
			name: "failed true",
			job: jobWithConditions(batchv1.JobCondition{
				Type:   batchv1.JobFailed,
				Status: corev1.ConditionTrue,
			}),
			want: true,
		},
		{
			name: "failed false",
			job: jobWithConditions(batchv1.JobCondition{
				Type:   batchv1.JobFailed,
				Status: corev1.ConditionFalse,
			}),
			want: false,
		},
		{
			name: "complete only",
			job: jobWithConditions(batchv1.JobCondition{
				Type:   batchv1.JobComplete,
				Status: corev1.ConditionTrue,
			}),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := isJobFailed(tt.job); got != tt.want {
				t.Fatalf("isJobFailed() = %t, want %t", got, tt.want)
			}
		})
	}
}

func jobWithConditions(conditions ...batchv1.JobCondition) *batchv1.Job {
	return &batchv1.Job{
		Status: batchv1.JobStatus{
			Conditions: conditions,
		},
	}
}
