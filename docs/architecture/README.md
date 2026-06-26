# Architecture

KRATOS is planned around five components:

1. Kubernetes manages resource lifecycle.
2. Volcano remains the final scheduler.
3. The CUDA Scheduling Operator evaluates workload history and cluster state.
4. The knowledge base stores workload profiles.
5. The profiler collects CUDA metrics after execution.

```mermaid
flowchart TB
    user[User] --> cr[CUDAExperiment]
    cr --> operator[CUDA Scheduling Operator]
    operator --> kb[Knowledge Base]
    operator --> cluster[Cluster Metrics]
    operator --> hints[NodeAffinity and NodeSelector]
    hints --> volcano[Volcano Scheduler]
    volcano --> kube[Kubernetes]
    kube --> workload[CUDA Workload]
    workload --> profiler[Profiler]
    profiler --> kb
```

The operator applies hard constraints first, then scores eligible nodes using
compute, memory, network, load, energy, and heterogeneity signals. Volcano still
performs the final scheduling decision.
