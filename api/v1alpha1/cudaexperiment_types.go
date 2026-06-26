package v1alpha1

// CudaExperimentSpec describes the desired state of a CUDA experiment.
//
// Kubebuilder should later wrap this spec in a full Kubernetes custom resource
// type with TypeMeta, ObjectMeta, Status, deepcopy generation, and CRD markers.
type CudaExperimentSpec struct {
	Model     string `json:"model"`
	Dataset   string `json:"dataset"`
	BatchSize int32  `json:"batchSize"`
	Epochs    int32  `json:"epochs"`
	Priority  string `json:"priority,omitempty"`
	Precision string `json:"precision,omitempty"`
}

// CudaExperimentStatus records the observed state of the experiment lifecycle.
type CudaExperimentStatus struct {
	Phase         string `json:"phase,omitempty"`
	WorkloadClass string `json:"workloadClass,omitempty"`
	ProfileName   string `json:"profileName,omitempty"`
	WorkflowName  string `json:"workflowName,omitempty"`
	VolcanoJob    string `json:"volcanoJob,omitempty"`
}
