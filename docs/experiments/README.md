# Experiment Guide

KRATOS exists to evaluate whether CUDA architectural profiling can improve
cloud-native scheduling decisions for GPU workloads.

## Workloads

Initial validation workloads should include:

- CIFAR-10
- CIFAR-100
- Tiny ImageNet
- ResNet50
- EfficientNet-B0
- Vision Transformer
- ConvNeXt Tiny

## Scheduling Scenarios

Compare KRATOS against progressively richer baselines:

1. Kubernetes default scheduler
2. Volcano FIFO
3. Volcano Fair Share
4. Volcano with MIG
5. Volcano with MIG and KRATOS GPU-aware policies

## Infrastructure Metrics

Track these metrics for each scenario:

- throughput
- makespan
- average waiting time
- average completion time
- GPU utilization
- fairness

## Architectural Metrics

Profiling and telemetry should capture:

- SM occupancy
- warp execution efficiency
- achieved FLOPS
- memory throughput
- memory bandwidth utilization
- L1 cache hit rate
- L2 cache hit rate
- GPU active time
- GPU idle time

## Results Organization

Keep experiment configuration, raw metrics, and analysis scripts separate:

- configuration: scheduler, queue, MIG, workload mix, and cluster setup
- raw metrics: Prometheus exports, MLflow runs, and profiling outputs
- analysis: notebooks or scripts that compute aggregate metrics and plots
