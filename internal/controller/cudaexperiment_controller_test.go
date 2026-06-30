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
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	gpuv1alpha1 "github.com/chirichexe/kratos/api/v1alpha1"
)

var _ = Describe("CUDAExperiment Controller", func() {
	Context("When reconciling a resource", func() {
		const (
			resourceName      = "test-resource"
			resourceNamespace = "default"
		)

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: resourceNamespace,
		}
		cudaexperiment := &gpuv1alpha1.CUDAExperiment{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind CUDAExperiment")
			err := k8sClient.Get(ctx, typeNamespacedName, cudaexperiment)
			if err != nil && errors.IsNotFound(err) {
				resource := &gpuv1alpha1.CUDAExperiment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: resourceNamespace,
					},
					Spec: gpuv1alpha1.CUDAExperimentSpec{
						Image:            "nvcr.io/nvidia/k8s/cuda-sample:vectoradd-cuda12.5.0",
						Command:          []string{"/cuda-samples/vectorAdd"},
						Arguments:        []string{"--iterations=1"},
						Replicas:         1,
						GPURequired:      1,
						RuntimeClassName: "nvidia",
						NumberOfGPUs:     1,
						ProfilingEnabled: true,
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &gpuv1alpha1.CUDAExperiment{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance CUDAExperiment")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should create an execution Job and update the resource status", func() {
			By("Reconciling the created resource")
			controllerReconciler := &CUDAExperimentReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the execution Job")
			job := &batchv1.Job{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      resourceName + executionJobNameSuffix,
				Namespace: resourceNamespace,
			}, job)
			Expect(err).NotTo(HaveOccurred())
			Expect(job.OwnerReferences).To(HaveLen(1))
			Expect(job.OwnerReferences[0].Name).To(Equal(resourceName))
			Expect(job.Spec.Template.Spec.RestartPolicy).To(Equal(corev1.RestartPolicyNever))
			Expect(job.Spec.Template.Spec.RuntimeClassName).NotTo(BeNil())
			Expect(*job.Spec.Template.Spec.RuntimeClassName).To(Equal("nvidia"))
			Expect(job.Spec.Template.Spec.Volumes).To(HaveLen(1))
			Expect(job.Spec.Template.Spec.Volumes[0].Name).To(Equal(sharedWorkloadVolumeName))
			Expect(job.Spec.Template.Spec.Containers).To(HaveLen(2))

			workload := job.Spec.Template.Spec.Containers[0]
			Expect(workload.Name).To(Equal("workload"))
			Expect(workload.Image).To(Equal("nvcr.io/nvidia/k8s/cuda-sample:vectoradd-cuda12.5.0"))
			Expect(workload.Command).To(Equal([]string{"/bin/sh", "-c", workloadStagingScript("/cuda-samples/vectorAdd")}))
			Expect(workload.VolumeMounts).To(ContainElement(corev1.VolumeMount{
				Name:      sharedWorkloadVolumeName,
				MountPath: sharedWorkloadMountPath,
			}))
			Expect(workload.Resources.Limits).NotTo(HaveKey(corev1.ResourceName("nvidia.com/gpu")))

			sidecar := job.Spec.Template.Spec.Containers[1]
			expectedExperiment := &gpuv1alpha1.CUDAExperiment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: resourceNamespace,
				},
				Spec: gpuv1alpha1.CUDAExperimentSpec{
					Arguments: []string{"--iterations=1"},
				},
			}
			Expect(sidecar.Name).To(Equal("nsight-compute-sidecar"))
			Expect(sidecar.Image).To(Equal(defaultNsightComputeImage))
			Expect(sidecar.ImagePullPolicy).To(Equal(corev1.PullIfNotPresent))
			Expect(sidecar.Command).To(Equal([]string{"/bin/sh", "-c", nsightComputeSidecarScript(expectedExperiment)}))
			Expect(sidecar.SecurityContext).NotTo(BeNil())
			Expect(sidecar.SecurityContext.Capabilities).NotTo(BeNil())
			Expect(sidecar.SecurityContext.Capabilities.Add).To(ContainElement(corev1.Capability("SYS_ADMIN")))
			Expect(sidecar.VolumeMounts).To(ContainElement(corev1.VolumeMount{
				Name:      sharedWorkloadVolumeName,
				MountPath: sharedWorkloadMountPath,
			}))
			gpuLimit := sidecar.Resources.Limits[corev1.ResourceName("nvidia.com/gpu")]
			Expect(gpuLimit.Value()).To(Equal(int64(1)))

			By("Verifying the CUDAExperiment status")
			err = k8sClient.Get(ctx, typeNamespacedName, cudaexperiment)
			Expect(err).NotTo(HaveOccurred())
			Expect(cudaexperiment.Status.ExecutionJobName).To(Equal(resourceName + executionJobNameSuffix))
			condition := meta.FindStatusCondition(cudaexperiment.Status.Conditions, executionJobConditionType)
			Expect(condition).NotTo(BeNil())
			Expect(condition.Reason).To(Equal("JobCreated"))
			Expect(condition.Status).To(Equal(metav1.ConditionTrue))
		})

		It("should keep the original single-container Job when profiling is disabled", func() {
			const disabledResourceName = "test-resource-no-profile"
			disabledNamespacedName := types.NamespacedName{
				Name:      disabledResourceName,
				Namespace: resourceNamespace,
			}
			resource := &gpuv1alpha1.CUDAExperiment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      disabledResourceName,
					Namespace: resourceNamespace,
				},
				Spec: gpuv1alpha1.CUDAExperimentSpec{
					Image:            "nvcr.io/nvidia/k8s/cuda-sample:vectoradd-cuda12.5.0",
					Command:          []string{"/cuda-samples/vectorAdd"},
					Arguments:        []string{"--iterations=1"},
					Replicas:         1,
					GPURequired:      1,
					RuntimeClassName: "nvidia",
					ProfilingEnabled: false,
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			defer func() {
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}()

			controllerReconciler := &CUDAExperimentReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: disabledNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			job := &batchv1.Job{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      disabledResourceName + executionJobNameSuffix,
				Namespace: resourceNamespace,
			}, job)
			Expect(err).NotTo(HaveOccurred())
			Expect(job.Spec.Template.Spec.Volumes).To(BeEmpty())
			Expect(job.Spec.Template.Spec.Containers).To(HaveLen(1))
			container := job.Spec.Template.Spec.Containers[0]
			Expect(container.Name).To(Equal("execution"))
			Expect(container.Image).To(Equal("nvcr.io/nvidia/k8s/cuda-sample:vectoradd-cuda12.5.0"))
			Expect(container.Command).To(Equal([]string{"/cuda-samples/vectorAdd"}))
			Expect(container.Args).To(Equal([]string{"--iterations=1"}))
			gpuLimit := container.Resources.Limits[corev1.ResourceName("nvidia.com/gpu")]
			Expect(gpuLimit.Value()).To(Equal(int64(1)))
		})
	})
})
