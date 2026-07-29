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
- A KRATOS controller image and Nsight Compute runner image available to the cluster.

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

Create the cluster using `nvkind`. By default, `nvkind` spins up a control-plane (master) node and a worker node with GPU access:

```bash
nvkind cluster create --name kratos-gpu
```

Alternatively, you can inspect or customize node declarations via [cluster/nvkind-cluster.yaml](../../cluster/nvkind-cluster.yaml).

Wait for all nodes to become ready:

```bash
kubectl wait --for=condition=Ready nodes --all --timeout=120s
kubectl get nodes
```

Verify that `nvkind` detects the host GPU on the worker node:

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

Wait for the device plugin rollout to finish:

```bash
kubectl rollout status daemonset nvdp-nvidia-device-plugin -n nvidia-device-plugin --timeout=60s
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
  "name": "kratos-gpu-worker",
  "capacity": "3",
  "allocatable": "3"
}
```

## Verify CUDA Scheduling

Create a test pod `gpu-vectoradd.yaml`:

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
kubectl wait --for=jsonpath='{.status.phase}'=Completed pod/gpu-vectoradd --timeout=60s
kubectl logs gpu-vectoradd
```

Expected output includes `Test PASSED`.

## Install KRATOS and Nsight Compute Runner

1. Install the `CUDAExperiment` and `WorkloadProfile` CRDs:

   ```bash
   make install
   kubectl get crd cudaexperiments.gpu.scheduler.io workloadprofiles.gpu.scheduler.io
   ```

2. Apply the profiling runner RBAC permissions:

   ```bash
   kubectl apply -f config/rbac/profiling_runner_configmap_role.yaml
   kubectl apply -f config/rbac/profiling_runner_configmap_role_binding.yaml
   ```

3. Build and load the controller image:

   ```bash
   make docker-build IMG=kratos-controller:v0.1.0
   kind load docker-image kratos-controller:v0.1.0 --name kratos-gpu
   ```

4. Build and load the Nsight Compute profiling runner image:

   ```bash
   cd test/nsight-compute-poc
   make build
   make load CLUSTER=kratos-gpu
   cd ../..
   ```

5. Deploy the controller manager:

   ```bash
   make deploy IMG=kratos-controller:v0.1.0
   kubectl rollout status deployment/kratos-controller-manager -n kratos-system
   ```

## Run a CUDAExperiment

Apply the sample custom resource:

```bash
kubectl apply -f config/samples/gpu_v1alpha1_cudaexperiment.yaml
kubectl get cudaexperiments.gpu.scheduler.io
```

Inspect the profiling and execution flow:

1. **Profiling Job**:
   ```bash
   kubectl get jobs
   kubectl logs job/cuda-vector-add-profiling
   ```
2. **Workload Profile & Summary ConfigMap**:
   ```bash
   kubectl get configmap cuda-vector-add-profile-summary -o yaml
   kubectl get workloadprofile cuda-vector-add-profile -o yaml
   ```
3. **Execution Job**:
   ```bash
   kubectl get job cuda-vector-add-execution
   kubectl logs job/cuda-vector-add-execution
   ```

The sample requests one GPU through `gpuRequired`. In a time-sliced local lab, that request consumes one advertised logical `nvidia.com/gpu` slot.

References:

- [nvkind](https://github.com/NVIDIA/nvkind)
- [NVIDIA device plugin time-slicing](https://github.com/NVIDIA/k8s-device-plugin#shared-access-to-gpus)
