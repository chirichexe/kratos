# Getting Started

KRATOS is a Go/Kubebuilder operator scaffold. It is not a complete deployable
GPU platform yet.

## Required Tools

- Go 1.23 or newer
- Docker or another OCI runtime
- kubectl and kustomize
- Kubebuilder and controller-gen
- Access to a Kubernetes cluster for integration work

The full experiment stack will also need the NVIDIA device plugin, Volcano,
Prometheus, Grafana, DCGM Exporter, Nsight Compute, and CUDA workload images.

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
kubebuilder init --domain scheduler.io --repo github.com/chirichexe/kratos
kubebuilder create api --group gpu --version v1alpha1 --kind CUDAExperiment --resource --controller
make manifests
make generate
```

For a local GPU-enabled lab, see [Local GPU Lab](kind-gpu.md).
