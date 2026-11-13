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
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	gpuv1alpha1 "github.com/chirichexe/kratos/api/v1alpha1"
	"github.com/chirichexe/kratos/internal/scoring"
)

// SetupCUDAExperimentWebhookWithManager registers the webhook for CUDAExperiment in the manager.
func SetupCUDAExperimentWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &gpuv1alpha1.CUDAExperiment{}).
		WithValidator(&CUDAExperimentValidator{}).
		Complete()
}

// +kubebuilder:webhook:path=/validate-gpu-scheduler-io-v1alpha1-cudaexperiment,mutating=false,failurePolicy=fail,sideEffects=None,groups=gpu.scheduler.io,resources=cudaexperiments,verbs=create;update,versions=v1alpha1,name=vcudaexperiment-v1alpha1.kb.io,admissionReviewVersions=v1

// CUDAExperimentValidator rejects spec.minimumComputeCapability /
// spec.minimumMemory values the reconciler could not otherwise parse until
// it actually tried to score nodes against them.
type CUDAExperimentValidator struct{}

func (v *CUDAExperimentValidator) ValidateCreate(_ context.Context, obj *gpuv1alpha1.CUDAExperiment) (admission.Warnings, error) {
	return nil, validateConstraints(obj)
}

func (v *CUDAExperimentValidator) ValidateUpdate(_ context.Context, _, obj *gpuv1alpha1.CUDAExperiment) (admission.Warnings, error) {
	return nil, validateConstraints(obj)
}

func (v *CUDAExperimentValidator) ValidateDelete(_ context.Context, _ *gpuv1alpha1.CUDAExperiment) (admission.Warnings, error) {
	return nil, nil
}

func validateConstraints(experiment *gpuv1alpha1.CUDAExperiment) error {
	if cc := experiment.Spec.MinimumComputeCapability; cc != "" {
		if _, _, err := scoring.ParseComputeCapability(cc); err != nil {
			return fmt.Errorf("spec.minimumComputeCapability: %w", err)
		}
	}
	if mem := experiment.Spec.MinimumMemory; mem != "" {
		if _, err := resource.ParseQuantity(mem); err != nil {
			return fmt.Errorf("spec.minimumMemory: %w", err)
		}
	}
	return nil
}
