# Getting Started

KRATOS is planned as a Kubebuilder-based Kubernetes operator written in Go. The
current repository contains the initial structure, package boundaries, and
documentation needed to start implementation.

## Prerequisites

Install these tools before working on the controller:

- Go 1.23 or newer
- Docker or another OCI-compatible container runtime
- kubectl
- kustomize
- Kubebuilder
- controller-gen
- Access to a Kubernetes cluster for integration tests

The full experimental environment will also require:

- NVIDIA GPU Operator
- Volcano
- Argo Workflows
- MLflow
- Prometheus, Grafana, and NVIDIA DCGM Exporter
- NVIDIA Nsight Compute and NVIDIA Nsight Systems for profiling work

## Repository Setup

Clone the repository and run the basic package check:

```bash
git clone git@github.com:chirichexe/kratos.git
cd kratos
go test ./...
```

If your environment prevents Go from writing to the default build cache, use a
writable cache path:

```bash
GOCACHE=/tmp/kratos-go-build-cache go test ./...
```

## Current State

The repository is a lightweight scaffold. It defines the intended module
hierarchy and placeholder packages, but the full Kubebuilder-generated
controller-runtime code has not been added yet.

Use this command sequence when replacing placeholders with generated operator
code in a fresh setup:

```bash
kubebuilder init --domain kratos.io --repo github.com/chirichexe/kratos
kubebuilder create api --group kratos --version v1alpha1 --kind AIExperiment --resource --controller
make manifests
make generate
```
