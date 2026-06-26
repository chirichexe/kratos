// Package v1alpha1 defines the KRATOS Kubernetes API.
//
// The main resource is CudaExperiment, a declarative abstraction for deep
// learning workloads. Users describe the logical experiment here while the
// controller derives Argo Workflows, Volcano Jobs, MLflow tracking metadata,
// GPU or MIG requirements, and monitoring labels.
package v1alpha1
