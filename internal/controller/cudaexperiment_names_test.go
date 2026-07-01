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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	gpuv1alpha1 "github.com/chirichexe/kratos/api/v1alpha1"
)

func TestCUDAExperimentNames(t *testing.T) {
	t.Parallel()

	experiment := &gpuv1alpha1.CUDAExperiment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vector-add",
			Namespace: "gpu-tests",
		},
	}

	tests := []struct {
		name string
		got  string
		want string
	}{
		{
			name: "execution job",
			got:  executionJobName(experiment),
			want: "vector-add-execution",
		},
		{
			name: "profiling job",
			got:  profilingJobName(experiment),
			want: "vector-add-profiling",
		},
		{
			name: "profile summary config map",
			got:  profileSummaryConfigMapName(experiment),
			want: "vector-add-profile-summary",
		},
		{
			name: "workload profile",
			got:  workloadProfileName(experiment),
			want: "vector-add-profile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.got != tt.want {
				t.Fatalf("name = %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestCUDAExperimentNamesAreStable(t *testing.T) {
	t.Parallel()

	experiment := &gpuv1alpha1.CUDAExperiment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "memory-scan",
			Namespace: "first-namespace",
		},
	}

	first := []string{
		executionJobName(experiment),
		profilingJobName(experiment),
		profileSummaryConfigMapName(experiment),
		workloadProfileName(experiment),
	}

	experiment.Namespace = "second-namespace"

	second := []string{
		executionJobName(experiment),
		profilingJobName(experiment),
		profileSummaryConfigMapName(experiment),
		workloadProfileName(experiment),
	}

	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("name changed after namespace update: before %q, after %q", first[i], second[i])
		}
	}
}

func TestCUDAExperimentNamesStayWithinDNS1123LabelLimitForCurrentScope(t *testing.T) {
	t.Parallel()

	experiment := &gpuv1alpha1.CUDAExperiment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
	}

	names := []string{
		executionJobName(experiment),
		profilingJobName(experiment),
		profileSummaryConfigMapName(experiment),
		workloadProfileName(experiment),
	}

	for _, name := range names {
		if len(name) > 63 {
			t.Fatalf("name %q has length %d, want no more than 63", name, len(name))
		}
	}
}
