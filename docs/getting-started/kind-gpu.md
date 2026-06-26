# kind GPU Lab

This setup is useful for local KRATOS development when you want a small
multi-node Kubernetes cluster and shared GPU capacity. It is a lab
approximation: kind nodes are containers on one host, and kind does not emulate
GPUs by itself. NVIDIA time-slicing shares a physical GPU by advertising
multiple logical GPU replicas, but it does not provide memory or fault
isolation between replicas.

## Prerequisites

- Linux host with NVIDIA driver and NVIDIA Container Toolkit installed.
- Docker configured with NVIDIA runtime support.
- `kind`, `kubectl`, and `helm`.

## Create a Multi-Node kind Cluster

Create `kind-gpu.yaml`:

```yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: kratos-gpu
nodes:
  - role: control-plane
  - role: worker
    labels:
      kratos.io/gpu-simulated-node: "true"
  - role: worker
    labels:
      kratos.io/gpu-simulated-node: "true"
  - role: worker
    labels:
      kratos.io/gpu-simulated-node: "true"
```

Start the cluster:

```bash
kind create cluster --config kind-gpu.yaml
kubectl get nodes
```

## Install the NVIDIA GPU Operator

Install the operator in its own namespace:

```bash
helm repo add nvidia https://helm.ngc.nvidia.com/nvidia
helm repo update
kubectl create namespace gpu-operator
helm install gpu-operator nvidia/gpu-operator -n gpu-operator
```

## Enable GPU Time-Slicing

Create `time-slicing-config.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: time-slicing-config
  namespace: gpu-operator
data:
  any: |-
    version: v1
    flags:
      migStrategy: none
    sharing:
      timeSlicing:
        renameByDefault: false
        failRequestsGreaterThanOne: false
        resources:
          - name: nvidia.com/gpu
            replicas: 4
```

Apply it and configure the device plugin:

```bash
kubectl apply -f time-slicing-config.yaml
kubectl patch clusterpolicies.nvidia.com/cluster-policy \
  -n gpu-operator --type merge \
  -p '{"spec":{"devicePlugin":{"config":{"name":"time-slicing-config","default":"any"}}}}'
```

The `replicas: 4` setting makes each visible physical GPU appear as four
schedulable logical GPU slots. For KRATOS experiments, use node labels to
simulate heterogeneous GPU nodes:

```bash
kubectl label node kratos-gpu-worker kratos.io/gpu-model=t4 kratos.io/gpu-memory=16Gi
kubectl label node kratos-gpu-worker2 kratos.io/gpu-model=a100 kratos.io/gpu-memory=40Gi
kubectl label node kratos-gpu-worker3 kratos.io/gpu-model=l4 kratos.io/gpu-memory=24Gi
```

Pods that request `nvidia.com/gpu` can run only on kind nodes where the NVIDIA
device plugin advertises GPU capacity. Labels on other nodes are still useful
for testing KRATOS scoring and filtering logic, but they do not create real GPU
devices.

## Verify

Check that the device plugin advertised shared GPU capacity:

```bash
kubectl describe node kratos-gpu-worker | grep -E "nvidia.com/gpu|replicas|SHARED"
kubectl get pods -n gpu-operator
```

If you change the time-slicing ConfigMap later, restart the device plugin:

```bash
kubectl rollout restart -n gpu-operator daemonset/nvidia-device-plugin-daemonset
```

References:

- [kind cluster configuration](https://kind.sigs.k8s.io/docs/user/configuration/)
- [NVIDIA GPU Operator time-slicing](https://docs.nvidia.com/datacenter/cloud-native/gpu-operator/latest/gpu-sharing.html)
