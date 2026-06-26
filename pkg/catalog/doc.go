// Package catalog stores and retrieves workload profiling results.
//
// Profiles are keyed by workload hash, usually derived from the image, command,
// and arguments. The catalog prevents repeated profiling for workloads that are
// already known.
package catalog
