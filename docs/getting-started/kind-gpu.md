# Local GPU Lab

This page describes an operational local setup for running KRATOS against a
Kubernetes cluster that can schedule CUDA workloads. It uses `nvkind` to create
a kind-based cluster with NVIDIA GPU support, then installs the NVIDIA device
plugin with time-slicing enabled.

Time-slicing advertises multiple logical `nvidia.com/gpu` slots for each
physical GPU. It is useful for local tests, but it does not provide memory or
fault isolation between workloads.

## Prerequisites

- Linux host with a working NVIDIA driver.
- Docker.
- NVIDIA Container Toolkit.
- `kind`, `kubectl`, `helm`, `jq`, and `nvkind`.
- A KRATOS controller image that is available to the cluster.

## Configure Docker GPU Support

Configure Docker to use the NVIDIA runtime:

```bash
sudo nvidia-ctk runtime configure --runtime=docker --set-as-default --cdi.enabled
sudo nvidia-ctk config --set accept-nvidia-visible-devices-as-volume-mounts=true --in-place
sudo systemctl restart docker
```

Verify that containers can access the GPU:

```bash
docker run --rm --gpus all nvidia/cuda:12.4.1-base-ubuntu22.04 nvidia-smi
```

## Create the GPU Cluster

Create the cluster:

```bash
nvkind cluster create --name kratos-gpu
```

Wait for the nodes to become ready:

```bash
kubectl wait --for=condition=Ready nodes --all --timeout=120s
kubectl get nodes
```

Check that `nvkind` can see the host GPU:

```bash
nvkind cluster print-gpus --name kratos-gpu
```

## Install GPU Time-Slicing

Add the NVIDIA device plugin Helm repository:

```bash
helm repo add nvdp https://nvidia.github.io/k8s-device-plugin
helm repo update
```

Create a device plugin configuration. This example exposes each physical GPU
as three schedulable logical GPU slots:

```bash
cat <<'EOF' > /tmp/nvidia-device-plugin-config.yaml
version: v1
flags:
  migStrategy: "none"
  failOnInitError: true
  nvidiaDriverRoot: "/"
  plugin:
    deviceListStrategy: envvar
    deviceIDStrategy: uuid
sharing:
  timeSlicing:
    failRequestsGreaterThanOne: true
    resources:
    - name: nvidia.com/gpu
      replicas: 3
EOF
```

Install the device plugin:

```bash
helm upgrade -i nvdp nvdp/nvidia-device-plugin \
  --namespace nvidia-device-plugin \
  --create-namespace \
  --set runtimeClassName=nvidia \
  --set config.default=config \
  --set-file config.map.config=/tmp/nvidia-device-plugin-config.yaml \
  --set affinity=null
```

Check the plugin pods:

```bash
kubectl get ds -n nvidia-device-plugin
kubectl get pods -n nvidia-device-plugin
```

Check the GPU capacity advertised by the nodes:

```bash
kubectl get nodes -o json | jq -r '
.items[] | {
  name: .metadata.name,
  capacity: .status.capacity["nvidia.com/gpu"],
  allocatable: .status.allocatable["nvidia.com/gpu"]
}'
```

A node with one physical GPU and `replicas: 3` should advertise:

```json
{
  "capacity": "3",
  "allocatable": "3"
}
```

## Verify CUDA Scheduling

Create `gpu-vectoradd.yaml`:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: gpu-vectoradd
spec:
  runtimeClassName: nvidia
  restartPolicy: Never
  containers:
  - name: cuda
    image: nvcr.io/nvidia/k8s/cuda-sample:vectoradd-cuda12.5.0
    resources:
      limits:
        nvidia.com/gpu: 1
```

Run the pod and inspect the result:

```bash
kubectl apply -f gpu-vectoradd.yaml
kubectl get pod gpu-vectoradd
kubectl logs gpu-vectoradd
```

## Install KRATOS

Install the `CUDAExperiment` CRD:

```bash
make install
kubectl get crd cudaexperiments.gpu.scheduler.io
```

If the image exists only on the local Docker daemon, load it into the cluster
before deploying:

```bash
kind load docker-image <local-image> --name kratos-gpu
```

Deploy the controller with an image that the cluster can pull or already has
loaded:

```bash
make deploy IMG=<registry-or-local-image>
```

Check the controller deployment:

```bash
kubectl get deployments -n kratos-system
kubectl get pods -n kratos-system
```

## Run a CUDAExperiment

Start from the sample custom resource:

```bash
kubectl apply -f config/samples/gpu_v1alpha1_cudaexperiment.yaml
kubectl get cudaexperiments
```

The sample requests one GPU through the `gpuRequired` field. In a time-sliced
local lab, that request consumes one advertised logical `nvidia.com/gpu` slot.

References:

- [nvkind](https://github.com/NVIDIA/nvkind)
- [NVIDIA device plugin time-slicing](https://github.com/NVIDIA/k8s-device-plugin#shared-access-to-gpus)
