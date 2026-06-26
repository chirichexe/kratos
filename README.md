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

---

**KRATOS** is an experimental platform designed as an academic project at the **University of Bologna** to study and optimize the execution of deep learning workloads in cloud-native environments based on Kubernetes. It addresses the common issues of GPU underutilization, resource fragmentation, and inefficient management of concurrent workloads in modern ML clusters.

## Project Goal

KRATOS will integrate advanced mechanisms for **GPU resource allocation**, **MLOps pipeline orchestration**, **CUDA workload profiling**, and **infrastructure monitoring** to evaluate the impact of different scheduling strategies. It leverages Kubernetes as the orchestration layer, **Volcano** for advanced batch scheduling and execution queue management, **Argo Workflows** with **MLflow** for training pipelines and experiment tracking, and NVIDIA tooling for GPU observability.

The central research goal is to connect CUDA microarchitectural workload behavior with cloud-native scheduling decisions. Instead of relying only on declared CPU, memory, and GPU requests, KRATOS classifies workloads from profiling data and uses that knowledge to guide GPU, MIG, and Volcano queue allocation.

## Core Features

- **GPU Sharing & Multi-Instance GPU (MIG)**: Integrates NVIDIA MIG technology to divide single physical GPUs into isolated instances, increasing resource utilization and system throughput.

- **Custom AIExperiment Operator**: Introduces an `AIExperiment` Kubernetes resource, abstracting the infrastructure complexity. It automatically provisions Argo workflows, Volcano jobs, and MLflow sessions based on declarative workload descriptions.

- **CUDA Workload Characterization**: Runs short profiling phases for unknown workloads using NVIDIA Nsight Compute and NVIDIA Nsight Systems. Metrics such as SM occupancy, warp efficiency, memory throughput, cache hit rate, and achieved FLOPS are stored in a persistent workload profiling catalog.

- **GPU-aware Scheduling Policies**: Classifies workloads as `compute-bound`, `memory-bound`, or `balanced` and evaluates policies such as complementary co-location, MIG profile selection, and Volcano queue assignment.

## Experimental Validation & Monitoring

Validation is conducted using standard ML benchmarks (e.g., MNIST, CIFAR, ResNet) to simulate multi-tenant scenarios. System performance and operational efficiency are measured across various scheduling and GPU partitioning configurations. Infrastructure monitoring is achieved via **Prometheus, Grafana, and NVIDIA DCGM Exporter** to gather detailed metrics for comparative analysis.

## AIExperiment Resource

Users interact with KRATOS through a single Kubernetes custom resource:

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

The user describes the logical experiment only. The operator is responsible for deriving Pods, Argo Workflows, Volcano Jobs, GPU or MIG requests, MLflow metadata, and monitoring configuration.

## Execution Flow

1. The user creates an `AIExperiment`.
2. The KRATOS operator receives the reconciliation event.
3. The operator checks whether a workload profile already exists.
4. If no profile exists, KRATOS runs a reduced profiling workload.
5. Profiling metrics are stored in the workload catalog.
6. The workload is classified as `compute-bound`, `memory-bound`, or `balanced`.
7. KRATOS selects a Volcano queue and GPU or MIG configuration.
8. The operator creates the Argo Workflow and Volcano Job.
9. Metrics and experiment metadata are exported to Prometheus and MLflow.

## Project Structure

The repository is structured as a Kubebuilder-based Kubernetes controller project:

```text
.
├── api/v1alpha1/              # AIExperiment API types and CRD schema source
├── cmd/manager/               # Controller manager entrypoint
├── config/                    # Kubebuilder and kustomize manifests
│   ├── crd/                   # Generated CustomResourceDefinitions
│   ├── default/               # Default kustomize installation entrypoint
│   ├── manager/               # Controller manager Deployment
│   ├── prometheus/            # Metrics and ServiceMonitor configuration
│   ├── rbac/                  # Generated controller permissions
│   └── samples/               # Example AIExperiment manifests
├── docs/architecture/         # System architecture notes
├── hack/                      # Generation boilerplate and helper scripts
├── internal/controller/       # AIExperiment reconciler implementation
├── pkg/catalog/               # Persistent workload profiling catalog
├── pkg/profiling/             # Nsight-based CUDA characterization logic
├── pkg/scheduling/            # GPU-aware scheduling policy decisions
├── pkg/telemetry/             # Prometheus architectural telemetry
├── pkg/volcano/               # Volcano Job and queue builders
├── pkg/workflow/              # Argo Workflow builders
└── test/e2e/                  # End-to-end controller tests
```

## Documentation

The documentation hierarchy starts at [docs/README.md](docs/README.md):

- [Getting Started](docs/getting-started/README.md): tools, setup, and basic verification.
- [Project Layout](docs/development/project-layout.md): module responsibilities and placement rules.
- [Development Workflow](docs/development/workflow.md): implementation, generation, testing, and commit flow.
- [Operator Guide](docs/operator/README.md): expected `AIExperiment` controller lifecycle.
- [Experiment Guide](docs/experiments/README.md): validation scenarios and metrics.
- [Architecture Notes](docs/architecture/README.md): system layers and research hypothesis.

## Kubebuilder Modules

- `api/v1alpha1`: defines the `AIExperiment` API surface, including experiment inputs and controller status.
- `internal/controller`: contains the reconciler that turns `AIExperiment` resources into profiling jobs, catalog records, Argo Workflows, Volcano Jobs, and status updates.
- `config/crd`, `config/rbac`, `config/manager`, and `config/prometheus`: contain generated deployment assets for the controller.
- `cmd/manager`: starts the controller-runtime manager and registers the API and reconciler.

The current scaffold is intentionally lightweight. In a fresh Kubebuilder setup, the equivalent initialization commands are:

```bash
kubebuilder init --domain kratos.io --repo github.com/chirichexe/kratos
kubebuilder create api --group kratos --version v1alpha1 --kind AIExperiment --resource --controller
```

After the generated controller-runtime code is in place, use:

```bash
make manifests
make generate
```

## Development Roadmap

1. **Infrastructure**: Kubernetes, NVIDIA GPU Operator, Volcano, Argo Workflows, MLflow, Prometheus, Grafana, and NVIDIA DCGM Exporter.
2. **KRATOS Operator**: `AIExperiment` CRD, reconciler, Volcano Job generation, and Argo Workflow generation.
3. **CUDA Profiling**: Nsight Compute and Nsight Systems integration for reduced workload characterization.
4. **Profiling Catalog**: persistent storage, workload classification, and architectural telemetry.
5. **GPU-aware Scheduling**: experimental policies and comparison against Kubernetes Scheduler, Volcano FIFO, Volcano Fair Share, Volcano with MIG, and Volcano with MIG plus KRATOS.

## Evaluation Metrics

Infrastructure metrics:

- Throughput
- Makespan
- Average waiting time
- Average completion time
- GPU utilization
- Fairness

Architectural metrics:

- SM occupancy
- Warp efficiency
- Memory throughput
- Cache hit rate
- Achieved FLOPS
