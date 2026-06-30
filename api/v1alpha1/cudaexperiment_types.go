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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CUDAExperimentSpec defines the desired state of CUDAExperiment.
type CUDAExperimentSpec struct {
	// image is the CUDA workload container image.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Image string `json:"image"`

	// command overrides the image entrypoint when set.
	// +optional
	Command []string `json:"command,omitempty"`

	// arguments are passed to the container entrypoint or command.
	// +optional
	Arguments []string `json:"arguments,omitempty"`

	// replicas is the number of Job pods to run.
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=1
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// gpuRequired is the number of NVIDIA GPUs requested by each pod.
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=1
	// +optional
	GPURequired int32 `json:"gpuRequired,omitempty"`

	// runtimeClassName selects the container runtime handler used for CUDA pods.
	// +kubebuilder:default="nvidia"
	// +kubebuilder:validation:MinLength=1
	// +optional
	RuntimeClassName string `json:"runtimeClassName,omitempty"`

	MinimumComputeCapability    string `json:"minimumComputeCapability,omitempty"`
	MinimumMemory               string `json:"minimumMemory,omitempty"`
	Priority                    string `json:"priority,omitempty"`
	ProfilingEnabled            bool   `json:"profilingEnabled,omitempty"`
	Distributed                 bool   `json:"distributed,omitempty"`
	NumberOfGPUs                int32  `json:"numberOfGPUs,omitempty"`
	NumberOfNodes               int32  `json:"numberOfNodes,omitempty"`
	MaxLatency                  string `json:"maxLatency,omitempty"`
	NetworkBandwidthRequirement string `json:"networkBandwidthRequirement,omitempty"`
}

// CUDAExperimentStatus defines the observed state of CUDAExperiment.
type CUDAExperimentStatus struct {
	// executionJobName is the name of the Kubernetes Job created for this experiment.
	// +optional
	ExecutionJobName string `json:"executionJobName,omitempty"`

	// conditions represent the current state of the CUDAExperiment resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// CUDAExperiment is the Schema for the cudaexperiments API
type CUDAExperiment struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of CUDAExperiment
	// +required
	Spec CUDAExperimentSpec `json:"spec"`

	// status defines the observed state of CUDAExperiment
	// +optional
	Status CUDAExperimentStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// CUDAExperimentList contains a list of CUDAExperiment
type CUDAExperimentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []CUDAExperiment `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &CUDAExperiment{}, &CUDAExperimentList{})
		return nil
	})
}
