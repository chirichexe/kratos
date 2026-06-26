# Project Layout

KRATOS follows the standard shape of a Kubebuilder project, with additional
domain packages for profiling, scheduling, telemetry, workflow generation, and
Volcano integration.

## Root Files

- `README.md`: project overview, research goal, and high-level structure.
- `PROJECT`: Kubebuilder project metadata.
- `go.mod`: Go module definition.
- `Makefile`: common development and generation commands.
- `LICENSE`: project license.

## Controller Modules

- `cmd/manager`: controller manager entrypoint.
- `api/v1alpha1`: `AIExperiment` API types and future CRD schema markers.
- `internal/controller`: reconcilers and controller-runtime wiring.
- `config/crd`: generated CustomResourceDefinition manifests.
- `config/rbac`: generated controller permissions.
- `config/manager`: controller manager deployment manifests.
- `config/default`: default kustomize installation entrypoint.
- `config/prometheus`: metrics and ServiceMonitor configuration.
- `config/samples`: sample custom resources for manual testing.

## KRATOS Domain Modules

- `pkg/catalog`: persistent workload profile lookup and storage.
- `pkg/profiling`: Nsight-based CUDA profiling and workload classification.
- `pkg/scheduling`: GPU-aware scheduling decisions.
- `pkg/telemetry`: Prometheus metrics for architectural workload data.
- `pkg/volcano`: Volcano Job and queue builders.
- `pkg/workflow`: Argo Workflow builders.

## Test and Support Directories

- `test/e2e`: end-to-end tests against a Kubernetes cluster.
- `hack`: boilerplate and helper scripts for generation.
- `docs`: project documentation.

## Placement Rules

- Put Kubernetes API types in `api/<version>`.
- Put reconciler code in `internal/controller`.
- Put reusable domain logic in `pkg`.
- Keep generated manifests under `config`.
- Keep experiment instructions, architecture notes, and contributor workflows in
  `docs`.
