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

**KRATOS** is an experimental platform designed as an academic projet at **University of Bologna** to study and optimize the execution of deep learning workloads in cloud-native environments based on Kubernetes. It addresses the common issues of GPU underutilization, resource fragmentation, and inefficient management of concurrent workloads in modern ML clusters.

## Project Goal

KRATOS will integrate advanced mechanisms for **GPU resource allocation**, **MLOps pipeline orchestration**, and **infrastructure monitoring** to evaluate the impact of different scheduling strategies. It leverages Kubernetes as the orchestration layer, utilizing **Volcano** for advanced batch scheduling and execution queue management, and **Argo Workflows** with **MLflow** for automating training pipelines, evaluation, and model registry.

## Core Features

- **GPU Sharing & Multi-Instance GPU (MIG)**: Integrates NVIDIA MIG technology to divide single physical GPUs into isolated instances, increasing resource utilization and system throughput.

- **Custom AIExperiment Operator**: Introduces an `AIExperiment` Kubernetes resource, abstracting the infrastructure complexity. It automatically provisions Argo workflows, Volcano jobs, and MLflow sessions based on declarative workload descriptions.

## Experimental Validation & Monitoring

Validation is conducted using standard ML benchmarks (e.g., MNIST, CIFAR, ResNet) to simulate multi-tenant scenarios. System performance and operational efficiency are measured across various scheduling and GPU partitioning configurations. Infrastructure monitoring is achieved via **Prometheus, Grafana, and NVIDIA DCGM Exporter** to gather detailed metrics for comparative analysis.
