# Experiments

KRATOS should evaluate whether workload history and CUDA metrics improve GPU
placement compared with resource-only scheduling.

## Baselines

Compare against:

1. Kubernetes default scheduler
2. Volcano FIFO
3. Volcano priority scheduling
4. Volcano fair sharing
5. Volcano with KRATOS node-selection hints

## Workload Classes

Initial classes:

- compute-bound
- memory-bound
- tensor-core-bound

Future classes may include communication-bound, IO-bound, and mixed workloads.

## Metrics

Track throughput, makespan, waiting time, completion time, GPU utilization,
memory utilization, active workloads, power consumption, temperature, and
profile reuse rate.

For distributed scenarios, also track bandwidth, latency, and topology effects.
