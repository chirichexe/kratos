# Project Structure

KRATOS follows a normal Kubebuilder layout plus small domain packages used by
the reconciler.

## Main Directories

- `api/v1alpha1`: `CudaExperiment` API types.
- `cmd/manager`: controller manager entrypoint.
- `internal/controller`: reconciliation logic.
- `config`: CRDs, RBAC, manager deployment, samples, and kustomize entrypoints.
- `pkg/catalog`: workload profile records.
- `pkg/profiling`: CUDA profiling and workload classification.
- `pkg/scheduling`: queue, GPU, MIG, and co-location decisions.
- `pkg/telemetry`: Prometheus metric helpers.
- `pkg/volcano`: Volcano object builders.
- `pkg/workflow`: Argo Workflow builders.
- `docs`: project notes.
- `test/e2e`: cluster-level tests.

Keep Kubernetes-facing code in `api`, `internal/controller`, and `config`.
Keep reusable profiling, scheduling, workflow, and telemetry logic in `pkg`.
