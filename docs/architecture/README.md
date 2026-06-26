# KRATOS Architecture

KRATOS is organized into five layers:

1. User layer: users submit only `AIExperiment` resources.
2. Orchestration layer: the KRATOS operator creates Argo Workflows, Volcano
   Jobs, MLflow metadata, GPU requests, and monitoring labels.
3. Scheduling layer: Volcano schedules the workload using queues and policies
   selected by KRATOS.
4. Profiling layer: Nsight Compute and Nsight Systems characterize unknown CUDA
   workloads through short profiling runs.
5. Monitoring layer: Prometheus, Grafana, and NVIDIA DCGM Exporter expose
   infrastructure and architectural metrics.

```mermaid
flowchart TB
    user[User] --> cr[AIExperiment]
    cr --> operator[KRATOS Operator]
    operator --> argo[Argo Workflows]
    operator --> volcano[Volcano Scheduler]
    operator --> mlflow[MLflow]
    operator --> metrics[Prometheus Metrics]
    volcano --> gpu[GPU or MIG Resources]
    metrics --> grafana[Grafana Dashboards]
```

## Reconciliation Flow

When an `AIExperiment` is created, the controller follows a small decision loop:

```mermaid
flowchart TD
    start[AIExperiment created] --> read[Read experiment spec]
    read --> lookup{Profile exists?}
    lookup -- yes --> classify[Use stored workload class]
    lookup -- no --> profile[Run reduced Nsight profiling]
    profile --> store[Store profile in catalog]
    store --> classify
    classify --> schedule[Select queue and GPU or MIG profile]
    schedule --> workflow[Create Argo Workflow]
    schedule --> job[Create Volcano Job]
    workflow --> observe[Export MLflow and Prometheus data]
    job --> observe
```

## Module Map

The codebase separates Kubernetes controller code from the domain modules used
by the reconciler:

```mermaid
flowchart LR
    api[api/v1alpha1] --> controller[internal/controller]
    controller --> catalog[pkg/catalog]
    controller --> profiling[pkg/profiling]
    controller --> scheduling[pkg/scheduling]
    controller --> workflow[pkg/workflow]
    controller --> volcano[pkg/volcano]
    controller --> telemetry[pkg/telemetry]
    manager[cmd/manager] --> controller
    config[config/] --> manager
```

The experimental hypothesis is that CUDA microarchitectural information can
improve GPU allocation, throughput, fairness, and makespan compared with
policies based only on declared resource requests.
