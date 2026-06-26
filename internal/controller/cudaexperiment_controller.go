package controller

// CudaExperimentReconciler is the placeholder for the Kubebuilder reconciler.
//
// Expected reconciliation flow:
// 1. Read the CudaExperiment custom resource.
// 2. Look up an existing workload profile in the catalog.
// 3. Trigger a short CUDA profiling workflow when no profile exists.
// 4. Classify the workload as compute-bound, memory-bound, or balanced.
// 5. Select Volcano queue, GPU or MIG shape, and co-location policy.
// 6. Create or update the Argo Workflow and Volcano Job.
// 7. Publish status, MLflow metadata, and Prometheus telemetry labels.
type CudaExperimentReconciler struct{}
