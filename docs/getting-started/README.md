# Getting Started

KRATOS is a Go/Kubebuilder operator scaffold. It is not a complete deployable
GPU platform yet.

## Required Tools

- Go 1.23 or newer
- Docker or another OCI runtime
- kubectl and kustomize
- Kubebuilder and controller-gen
- Access to a Kubernetes cluster for integration work

The full experiment stack will also need NVIDIA GPU Operator, Volcano, Argo
Workflows, MLflow, Prometheus, Grafana, DCGM Exporter, Nsight Compute, and
Nsight Systems.

## Local Check

```bash
git clone git@github.com:chirichexe/kratos.git
cd kratos
go test ./...
```

If the default Go build cache is not writable:

```bash
GOCACHE=/tmp/kratos-go-build-cache go test ./...
```

## Kubebuilder Setup

Use this sequence when regenerating the scaffold from a fresh operator setup:

```bash
kubebuilder init --domain kratos.io --repo github.com/chirichexe/kratos
kubebuilder create api --group kratos --version v1alpha1 --kind CudaExperiment --resource --controller
make manifests
make generate
```
