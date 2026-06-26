# Architecture

KRATOS is planned around five layers:

1. Users submit `CudaExperiment` resources.
2. The operator creates profiling, workflow, scheduling, metadata, and metrics
   resources.
3. Volcano schedules workloads onto GPU or MIG capacity.
4. Nsight tooling profiles unknown CUDA workloads.
5. Prometheus, Grafana, and DCGM Exporter expose metrics.

```mermaid
flowchart TB
    user[User] --> cr[CudaExperiment]
    cr --> operator[KRATOS Operator]
    operator --> argo[Argo Workflows]
    operator --> volcano[Volcano Scheduler]
    operator --> mlflow[MLflow]
    operator --> metrics[Prometheus]
    volcano --> gpu[GPU or MIG]
    metrics --> grafana[Grafana]
```

The research hypothesis is that CUDA microarchitectural data can improve GPU
allocation, throughput, fairness, and makespan compared with policies based
only on declared resource requests.
