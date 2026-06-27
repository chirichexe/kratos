# GPU Kernel Test App

This is a minimal dockerized CUDA workload for smoke-testing GPU scheduling and
runtime access. The application launches a simple vector-add kernel, copies the
result back to host memory, validates it, and exits non-zero if CUDA execution
fails.

## Build

```bash
docker build -t kratos-gpu-kernel-test:latest .
```

Or:

```bash
make build
```

## Run Locally

Requires a host with NVIDIA drivers and the NVIDIA Container Toolkit.

```bash
docker run --rm --gpus all kratos-gpu-kernel-test:latest
```

Or:

```bash
make run
```

Expected output:

```text
Launching vectorAdd kernel with 4096 elements
GPU kernel completed successfully
Validation passed
```

## Kubernetes Job Example

Apply the included manifest after the image is available to the cluster:

```bash
kubectl apply -f job.yaml
```

Manifest contents:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: gpu-kernel-test
spec:
  template:
    spec:
      restartPolicy: Never
      containers:
        - name: gpu-kernel
          image: kratos-gpu-kernel-test:latest
          imagePullPolicy: IfNotPresent
          resources:
            limits:
              nvidia.com/gpu: 1
  backoffLimit: 0
```
