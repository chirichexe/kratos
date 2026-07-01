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
	"slices"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	gpuv1alpha1 "github.com/chirichexe/kratos/api/v1alpha1"
)

const testCUDAImage = "nvcr.io/nvidia/k8s/cuda-sample:vectoradd-cuda12.5.0"

func TestExecutionJobForExperimentBuildsNormalWorkloadJob(t *testing.T) {
	t.Parallel()

	experiment := cudaExperimentForJobBuilderTest(false)
	reconciler := &CUDAExperimentReconciler{
		NsightComputeImage: "custom/nsight:latest",
	}

	job := reconciler.executionJobForExperiment(experiment)

	if job.Name != "vector-add-execution" {
		t.Fatalf("execution job name = %q, want %q", job.Name, "vector-add-execution")
	}
	if len(job.Spec.Template.Spec.InitContainers) != 0 {
		t.Fatalf("execution job init containers = %d, want 0", len(job.Spec.Template.Spec.InitContainers))
	}
	if len(job.Spec.Template.Spec.Volumes) != 0 {
		t.Fatalf("execution job volumes = %d, want 0", len(job.Spec.Template.Spec.Volumes))
	}
	if len(job.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("execution job containers = %d, want 1", len(job.Spec.Template.Spec.Containers))
	}

	container := job.Spec.Template.Spec.Containers[0]
	if container.Name != "execution" {
		t.Fatalf("execution container name = %q, want %q", container.Name, "execution")
	}
	if container.Image != experiment.Spec.Image {
		t.Fatalf("execution container image = %q, want %q", container.Image, experiment.Spec.Image)
	}
	if container.Image == reconciler.nsightComputeImage() {
		t.Fatalf("execution container unexpectedly uses Nsight Compute image %q", container.Image)
	}
}

func TestProfilingJobForExperimentBuildsNsightComputeJob(t *testing.T) {
	t.Parallel()

	experiment := cudaExperimentForJobBuilderTest(true)
	reconciler := &CUDAExperimentReconciler{
		NsightComputeImage: "custom/nsight:latest",
	}

	job := reconciler.profilingJobForExperiment(experiment)

	if job.Name != "vector-add-profiling" {
		t.Fatalf("profiling job name = %q, want %q", job.Name, "vector-add-profiling")
	}
	if len(job.Spec.Template.Spec.InitContainers) != 1 {
		t.Fatalf("profiling job init containers = %d, want 1", len(job.Spec.Template.Spec.InitContainers))
	}
	if job.Spec.Template.Spec.InitContainers[0].Name != "stage-workload" {
		t.Fatalf("profiling init container name = %q, want %q", job.Spec.Template.Spec.InitContainers[0].Name, "stage-workload")
	}
	if len(job.Spec.Template.Spec.Volumes) != 1 {
		t.Fatalf("profiling job volumes = %d, want 1", len(job.Spec.Template.Spec.Volumes))
	}
	if job.Spec.Template.Spec.Volumes[0].Name != sharedWorkloadVolumeName {
		t.Fatalf("profiling volume name = %q, want %q", job.Spec.Template.Spec.Volumes[0].Name, sharedWorkloadVolumeName)
	}
	if len(job.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("profiling job containers = %d, want 1", len(job.Spec.Template.Spec.Containers))
	}

	container := job.Spec.Template.Spec.Containers[0]
	if container.Name != "profiling-runner" {
		t.Fatalf("profiling container name = %q, want %q", container.Name, "profiling-runner")
	}
	if container.Image != reconciler.nsightComputeImage() {
		t.Fatalf("profiling container image = %q, want %q", container.Image, reconciler.nsightComputeImage())
	}

	wantCommand := []string{"/scripts/profile.sh", sharedWorkloadMountPath + "/workload", profilingReportPath(experiment)}
	if !slices.Equal(container.Command, wantCommand) {
		t.Fatalf("profiling command = %q, want %q", container.Command, wantCommand)
	}
	assertEnvVar(t, container, "KRATOS_EXPERIMENT_NAME", experiment.Name)
	assertEnvVar(t, container, "KRATOS_EXPERIMENT_NAMESPACE", experiment.Namespace)
	assertEnvVar(t, container, "KRATOS_PROFILE_SUMMARY_CONFIGMAP", profileSummaryConfigMapName(experiment))
	assertEnvVar(t, container, "KRATOS_PROFILE_SUMMARY_KEY", profileSummaryConfigMapKey)
	if !containerMountsVolume(container, sharedWorkloadVolumeName) {
		t.Fatalf("profiling container does not mount shared workload volume %q", sharedWorkloadVolumeName)
	}
	if !containerMountsVolume(job.Spec.Template.Spec.InitContainers[0], sharedWorkloadVolumeName) {
		t.Fatalf("profiling init container does not mount shared workload volume %q", sharedWorkloadVolumeName)
	}
}

func cudaExperimentForJobBuilderTest(profilingEnabled bool) *gpuv1alpha1.CUDAExperiment {
	return &gpuv1alpha1.CUDAExperiment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vector-add",
			Namespace: "default",
		},
		Spec: gpuv1alpha1.CUDAExperimentSpec{
			Image:            testCUDAImage,
			Command:          []string{defaultWorkloadExecutable},
			Arguments:        []string{"--iterations=1"},
			Replicas:         1,
			GPURequired:      1,
			RuntimeClassName: defaultRuntimeClassName,
			ProfilingEnabled: profilingEnabled,
		},
	}
}

func assertEnvVar(t *testing.T, container corev1.Container, name, value string) {
	t.Helper()

	for _, env := range container.Env {
		if env.Name == name {
			if env.Value != value {
				t.Fatalf("env %s = %q, want %q", name, env.Value, value)
			}
			return
		}
	}
	t.Fatalf("env %s is missing", name)
}

func containerMountsVolume(container corev1.Container, volumeName string) bool {
	for _, mount := range container.VolumeMounts {
		if mount.Name == volumeName {
			return true
		}
	}
	return false
}
