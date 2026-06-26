// Package catalog stores and retrieves workload profiling results.
//
// Profiles are keyed by model, dataset, batch size, and precision. The catalog
// prevents repeated Nsight profiling for workloads that are already known.
package catalog
