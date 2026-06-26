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

KRATOS is an academic Kubernetes operator project for studying how CUDA deep
learning workloads can be scheduled more efficiently on GPU clusters.

The main idea is to connect CUDA profiling data with cloud-native scheduling.
Instead of relying only on declared CPU, memory, and GPU requests, KRATOS plans
to classify workloads as `compute-bound`, `memory-bound`, or `balanced`, then
use that information for Volcano queue selection, GPU or MIG placement, and
experiment tracking.

## Status

The repository is currently a lightweight Kubebuilder-style scaffold. It
defines the `CudaExperiment` API direction, controller boundary, and domain
packages for profiling, scheduling, telemetry, Argo workflow generation,
Volcano integration, and profile catalog storage.

Planned integrations include:

- Kubernetes and Volcano for batch orchestration.
- NVIDIA MIG, DCGM Exporter, Nsight Compute, and Nsight Systems for GPU
  partitioning, monitoring, and profiling.
- Argo Workflows and MLflow for training pipelines and experiment metadata.
- Prometheus and Grafana for infrastructure metrics.

## CudaExperiment

Users describe a training run with one Kubernetes custom resource:

```yaml
apiVersion: kratos.io/v1alpha1
kind: CudaExperiment
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

## Documentation

- [Project structure](docs/development/project-layout.md)
- [Getting started](docs/getting-started/README.md)
- [Architecture](docs/architecture/README.md)
- [Operator lifecycle](docs/operator/README.md)
- [Experiment notes](docs/experiments/README.md)
