# Project Structure

KRATOS follows a normal Kubebuilder layout plus small domain packages used by
the operator.

## Main Directories

- `api/v1alpha1`: `CUDAExperiment` API types.
- `cmd/manager`: controller manager entrypoint.
- `internal/controller`: reconciliation logic.
- `config`: CRDs, RBAC, manager deployment, samples, and kustomize entrypoints.
- `pkg/catalog`: workload profile keys and profile records.
- `pkg/profiling`: CUDA metric collection and workload classification.
- `pkg/scheduling`: hard constraints, scoring, and node-selection decisions.
- `pkg/telemetry`: Prometheus metric helpers.
- `pkg/volcano`: Volcano resource builders.
- `pkg/workflow`: workload construction helpers.
- `docs`: project notes.
- `test/e2e`: cluster-level tests.

Keep Kubernetes-facing code in `api`, `internal/controller`, and `config`.
Keep reusable profiling, catalog, scheduling, Volcano, and telemetry logic in
`pkg`.
