package catalog

// WorkloadKey uniquely identifies a profiled training workload.
type WorkloadKey struct {
	Model     string
	Dataset   string
	BatchSize int32
	Precision string
}

// Profile contains the architectural metrics used by KRATOS policies.
type Profile struct {
	WorkloadClass      string
	SMOccupancy        float64
	WarpEfficiency     float64
	AchievedFLOPS      float64
	MemoryThroughput   float64
	MemoryBandwidth    float64
	L1CacheHitRate     float64
	L2CacheHitRate     float64
	GPUActiveTimeRatio float64
}
