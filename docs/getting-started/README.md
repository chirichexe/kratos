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
make test
```

The `make test` target prepares envtest binaries before running controller
tests. If the default Go build cache is not writable, override it:

```bash
GOCACHE=/tmp/kratos-go-build-cache make test
```

## Run Locally

Install the `CUDAExperiment` CRD into the current Kubernetes context:

```bash
make install
```

Run the controller from your host:

```bash
make run
```

Apply the sample custom resource from another terminal:

```bash
kubectl apply -f config/samples/gpu_v1alpha1_cudaexperiment.yaml
kubectl get cudaexperiments.gpu.scheduler.io
```

## Submit a CUDA Experiment

A `CUDAExperiment` describes the CUDA image to run and the GPU resources needed
by each Job pod. The minimal fields are:

```yaml
apiVersion: gpu.scheduler.io/v1alpha1
kind: CUDAExperiment
metadata:
  name: cuda-vector-add
spec:
  image: nvcr.io/nvidia/k8s/cuda-sample:vectoradd-cuda12.5.0
  runtimeClassName: nvidia
  replicas: 1
  gpuRequired: 1
```

`runtimeClassName` defaults to `nvidia`, which is required by the local GPU
Kind setup so the pod starts through the NVIDIA runtime handler. Set `command`
and `arguments` only when the image entrypoint is not enough for your workload.

Submit and inspect an experiment:

```bash
kubectl apply -f config/samples/gpu_v1alpha1_cudaexperiment.yaml
kubectl get cudaexperiment cuda-vector-add -o yaml
kubectl get job cuda-vector-add-execution
kubectl get pods -l gpu.scheduler.io/experiment=cuda-vector-add
kubectl logs job/cuda-vector-add-execution
```

The controller creates a Job named `<experiment-name>-execution` and records
that name in `status.executionJobName`. To rerun the same experiment name,
delete the generated Job or create a new `CUDAExperiment` with a different
name.

## Kubebuilder Setup

Use this sequence when regenerating the scaffold from a fresh operator setup:

```bash
kubebuilder init --domain scheduler.io --repo github.com/chirichexe/kratos
kubebuilder create api --group gpu --version v1alpha1 --kind CUDAExperiment --resource --controller
make manifests
make generate
```

For a local GPU-enabled lab, see [Local GPU Lab](kind-gpu.md).
