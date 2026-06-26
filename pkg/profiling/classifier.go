package profiling

// WorkloadClass is the scheduling category derived from profiling metrics.
type WorkloadClass string

const (
	// ComputeBound marks workloads dominated by arithmetic throughput.
	ComputeBound WorkloadClass = "compute-bound"
	// MemoryBound marks workloads dominated by memory bandwidth or cache limits.
	MemoryBound WorkloadClass = "memory-bound"
	// TensorCoreBound marks workloads dominated by Tensor Core utilization.
	TensorCoreBound WorkloadClass = "tensor-core-bound"
)
