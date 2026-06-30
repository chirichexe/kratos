# KRATOS Documentation

KRATOS is an academic Kubernetes operator project for studying
application-aware GPU scheduling of CUDA workloads on heterogeneous clusters.

The current goal is to let users describe CUDA workloads with requirements such
as GPU memory, compute capability, priority, replica count, and distributed
constraints. The controller can then use profiling information from previous
runs to score eligible nodes for later executions.

## Pages

- [Getting Started](getting-started/README.md): local setup and quick checks.
- [Local GPU Lab](getting-started/kind-gpu.md): GPU-enabled local cluster with
  NVIDIA time-slicing.
- [Architecture](architecture/README.md): planned control flow and system
  components.
- [Operator](operator/README.md): expected `CUDAExperiment` lifecycle.
- [Observability](observability/README.md): local Prometheus and Grafana stack.
- [Experiments](experiments/README.md): baselines, workload classes, and
  metrics.
- [Development Workflow](development/workflow.md): common development commands.
- [Project Structure](development/project-layout.md): repository layout.
