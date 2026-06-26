<div align="center">

# KRATOS

**Kubernetes Resource-aware Autonomous Training and Orchestration System**

[![Kubernetes](https://img.shields.io/badge/kubernetes-%23326ce5.svg?style=for-the-badge&logo=kubernetes&logoColor=white)](https://kubernetes.io/)
[![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)](https://go.dev/)
[![MLflow](https://img.shields.io/badge/MLflow-0194E2.svg?style=for-the-badge&logo=MLflow&logoColor=white)](https://mlflow.org/)
[![NVIDIA](https://img.shields.io/badge/NVIDIA-76B900.svg?style=for-the-badge&logo=nVIDIA&logoColor=white)](https://www.nvidia.com/)
[![Prometheus](https://img.shields.io/badge/Prometheus-E6522C?style=for-the-badge&logo=prometheus&logoColor=white)](https://prometheus.io/)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg?style=for-the-badge)](https://opensource.org/licenses/Apache-2.0)

</div>

KRATOS is an academic Kubernetes operator project for studying how deep
learning workloads can be scheduled more efficiently on GPU clusters.

The main idea is to connect CUDA profiling data with cloud-native scheduling.
Instead of relying only on declared CPU, memory, and GPU requests, KRATOS plans
to classify workloads as `compute-bound`, `memory-bound`, or `balanced`, then
use that information for Volcano queue selection, GPU or MIG placement, and
experiment tracking.

## Scope

KRATOS is currently a lightweight Kubebuilder-style scaffold. The repository
defines the API shape, controller boundary, and domain packages for profiling,
scheduling, telemetry, Argo workflow generation, Volcano integration, and
profile catalog storage.

Planned integrations include:

- Kubernetes and Volcano for batch orchestration.
- NVIDIA MIG, DCGM Exporter, Nsight Compute, and Nsight Systems for GPU
  partitioning, monitoring, and profiling.
- Argo Workflows and MLflow for training pipelines and experiment metadata.
- Prometheus and Grafana for infrastructure metrics.

## AIExperiment

Users describe a training run with one Kubernetes custom resource:

```yaml
apiVersion: kratos.io/v1alpha1
kind: AIExperiment
metadata:
  name: resnet50-cifar100
spec:
  model: resnet50
  dataset: cifar100
  batchSize: 128
  epochs: 50
  priority: medium
  precision: fp32
```

The operator is responsible for translating that intent into profiling jobs,
Argo Workflows, Volcano Jobs, MLflow metadata, GPU or MIG choices, and status
updates.

## Repository Layout

```text
api/v1alpha1/         AIExperiment API types
cmd/manager/          controller manager entrypoint
config/               generated and kustomize manifests
internal/controller/  AIExperiment reconciler
pkg/catalog/          workload profile records
pkg/profiling/        CUDA profiling and classification
pkg/scheduling/       GPU-aware scheduling policies
pkg/telemetry/        Prometheus metric helpers
pkg/volcano/          Volcano resource builders
pkg/workflow/         Argo Workflow builders
docs/                 short project notes
test/e2e/             end-to-end test placeholder
```

## Development

Run the package tests:

```bash
go test ./...
```

Regenerate Kubernetes assets after API or RBAC changes:

```bash
make manifests
make generate
```

More notes are in [docs/README.md](docs/README.md).
