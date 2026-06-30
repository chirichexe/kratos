# Observability

KRATOS uses Prometheus, Grafana, and DCGM Exporter to inspect GPU runtime
metrics during local experiments.

This setup is intended for the local `nvkind` lab. Production clusters should
review chart values, storage, ingress, authentication, and retention settings.

## GPU Observability Setup

Install the Prometheus stack:

```bash
kubectl create namespace monitoring

helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

helm install kube-prometheus-stack prometheus-community/kube-prometheus-stack \
  -n monitoring
```

Install DCGM Exporter:

```bash
helm repo add dcgm-exporter https://nvidia.github.io/dcgm-exporter/helm-charts
helm repo update

helm install dcgm-exporter dcgm-exporter/dcgm-exporter \
  -n monitoring \
  -f cluster/dcgm-values.yaml
```

The values file pins DCGM Exporter to the GPU node, uses the NVIDIA runtime,
and requests one GPU:

```yaml
nodeSelector:
  accelerator: nvidia

runtimeClassName: nvidia

resources:
  limits:
    nvidia.com/gpu: 1
  requests:
    cpu: 100m
    memory: 128Mi
```

For `nvkind`, `runtimeClassName: nvidia` is required. Without it, the exporter
pod may start without the expected NVIDIA container environment and crash.

## GPU Node Label

Find the node with GPU capacity:

```bash
kubectl get nodes -o custom-columns=NAME:.metadata.name,GPU:.status.allocatable.nvidia\\.com/gpu
```

Label that node so DCGM Exporter is scheduled there:

```bash
kubectl label node kratos-gpu-worker accelerator=nvidia --overwrite
```

## ServiceMonitor

Prometheus from `kube-prometheus-stack` selects ServiceMonitors with the release
label. Add it to the DCGM ServiceMonitor:

```bash
kubectl label servicemonitor dcgm-exporter \
  -n monitoring \
  release=kube-prometheus-stack \
  --overwrite
```

## Verify

Check the monitoring pods and services:

```bash
kubectl get pods -n monitoring
kubectl get svc -n monitoring
```

Port-forward Prometheus:

```bash
kubectl port-forward -n monitoring svc/kube-prometheus-stack-prometheus 9090:9090
```

Query GPU metrics:

```bash
curl "http://localhost:9090/api/v1/query?query=DCGM_FI_DEV_GPU_UTIL"
curl "http://localhost:9090/api/v1/query?query=DCGM_FI_DEV_FB_USED"
curl "http://localhost:9090/api/v1/query?query=DCGM_FI_DEV_POWER_USAGE"
```

Expected metrics include GPU utilization, framebuffer memory usage, and power
usage. `DCGM_FI_DEV_GPU_UTIL` can remain `0` if the workload is too short.

DCGM Exporter also exposes raw metrics on its `/metrics` endpoint.

## Access Grafana Locally

```bash
kubectl port-forward -n monitoring svc/kube-prometheus-stack-grafana 3000:80
```

Open `http://localhost:3000`.

## Remove

```bash
helm uninstall dcgm-exporter -n monitoring
helm uninstall kube-prometheus-stack -n monitoring
kubectl delete namespace monitoring
```
