package v1alpha1

// CUDAExperimentSpec describes the desired state of a CUDA workload.
//
// Kubebuilder should later wrap this spec in a full Kubernetes custom resource
// type with TypeMeta, ObjectMeta, Status, deepcopy generation, and CRD markers.
type CUDAExperimentSpec struct {
	Image                       string   `json:"image"`
	Command                     []string `json:"command,omitempty"`
	Arguments                   []string `json:"arguments,omitempty"`
	Replicas                    int32    `json:"replicas,omitempty"`
	GPURequired                 int32    `json:"gpuRequired,omitempty"`
	MinimumComputeCapability    string   `json:"minimumComputeCapability,omitempty"`
	MinimumMemory               string   `json:"minimumMemory,omitempty"`
	Priority                    string   `json:"priority,omitempty"`
	ProfilingEnabled            bool     `json:"profilingEnabled,omitempty"`
	Distributed                 bool     `json:"distributed,omitempty"`
	NumberOfGPUs                int32    `json:"numberOfGPUs,omitempty"`
	NumberOfNodes               int32    `json:"numberOfNodes,omitempty"`
	MaxLatency                  string   `json:"maxLatency,omitempty"`
	NetworkBandwidthRequirement string   `json:"networkBandwidthRequirement,omitempty"`
}

// CUDAExperimentStatus records the observed state of the scheduling lifecycle.
type CUDAExperimentStatus struct {
	State            string   `json:"state,omitempty"`
	SelectedNode     string   `json:"selectedNode,omitempty"`
	WorkloadProfile  string   `json:"workloadProfile,omitempty"`
	ExecutionHistory []string `json:"executionHistory,omitempty"`
}
