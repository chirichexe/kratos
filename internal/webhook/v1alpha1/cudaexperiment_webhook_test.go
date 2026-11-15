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

package v1alpha1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	gpuv1alpha1 "github.com/chirichexe/kratos/api/v1alpha1"
)

const malformedMemory = "lots"

var _ = Describe("CUDAExperiment Webhook", func() {
	var (
		obj       *gpuv1alpha1.CUDAExperiment
		validator CUDAExperimentValidator
	)

	BeforeEach(func() {
		obj = &gpuv1alpha1.CUDAExperiment{}
		validator = CUDAExperimentValidator{}
	})

	It("admits a CUDAExperiment with no constraints set", func() {
		_, err := validator.ValidateCreate(ctx, obj)
		Expect(err).NotTo(HaveOccurred())
	})

	It("admits valid minimumComputeCapability and minimumMemory", func() {
		obj.Spec.MinimumComputeCapability = "7.0"
		obj.Spec.MinimumMemory = "4Gi"
		_, err := validator.ValidateCreate(ctx, obj)
		Expect(err).NotTo(HaveOccurred())
	})

	It("rejects a malformed minimumComputeCapability", func() {
		obj.Spec.MinimumComputeCapability = "fast"
		_, err := validator.ValidateCreate(ctx, obj)
		Expect(err).To(HaveOccurred())
	})

	It("rejects a malformed minimumMemory", func() {
		obj.Spec.MinimumMemory = malformedMemory
		_, err := validator.ValidateCreate(ctx, obj)
		Expect(err).To(HaveOccurred())
	})

	It("validates updates the same way as creates", func() {
		oldObj := &gpuv1alpha1.CUDAExperiment{}
		obj.Spec.MinimumMemory = malformedMemory
		_, err := validator.ValidateUpdate(ctx, oldObj, obj)
		Expect(err).To(HaveOccurred())
	})

	It("never rejects a delete", func() {
		obj.Spec.MinimumMemory = malformedMemory
		_, err := validator.ValidateDelete(ctx, obj)
		Expect(err).NotTo(HaveOccurred())
	})
})
