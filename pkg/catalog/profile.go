package catalog

// WorkloadKey uniquely identifies a profiled CUDA workload.
type WorkloadKey struct {
	WorkloadHash string
}

// Profile contains the historical metrics used by KRATOS scheduling policies.
type Profile struct {
	Classification          string
	ComputeUtilization      float64
	MemoryUtilization       float64
	TensorUtilization       float64
	AverageExecutionTime    float64
	AveragePowerConsumption float64
	LastUpdate              string
}
