# Experiments

KRATOS should evaluate whether CUDA profiling improves GPU scheduling compared
with standard Kubernetes and Volcano policies.

## Workloads

Start with common vision workloads such as CIFAR-10, CIFAR-100, Tiny ImageNet,
ResNet50, EfficientNet-B0, Vision Transformer, and ConvNeXt Tiny.

## Baselines

Compare against:

1. Kubernetes default scheduler
2. Volcano FIFO
3. Volcano Fair Share
4. Volcano with MIG
5. Volcano with MIG and KRATOS policies

## Metrics

Track throughput, makespan, waiting time, completion time, GPU utilization,
fairness, SM occupancy, warp efficiency, memory throughput, cache hit rate,
achieved FLOPS, GPU active time, and GPU idle time.

Keep configuration, raw metrics, and analysis scripts separate.
