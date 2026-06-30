# Nsight Compute Kubernetes Proof of Concept

This directory contains a focused proof of concept for running NVIDIA Nsight
Compute inside the GPU-enabled Kubernetes cluster used by KRATOS. It does not
integrate with `CUDAExperiment`, the controller, workload classification, or any
parser. The only purpose is to prove that a Kubernetes `Job` can request a GPU,
start a CUDA workload, and collect Nsight Compute metrics.

## Approach

The most direct execution path is a plain Kubernetes `Job` with:

- `runtimeClassName: nvidia`, matching the local `nvkind` setup documented in
  `docs/getting-started/kind-gpu.md`.
- a GPU limit of `nvidia.com/gpu: 1`, matching the controller-created Jobs and
  the existing CUDA smoke tests.
- a container image that includes CUDA and the Nsight Compute CLI (`ncu`).
- an init container using NVIDIA's cached vectorAdd sample image to copy
  `/cuda-samples/vectorAdd` into a shared `emptyDir`.
- `SYS_ADMIN` capability, because NVIDIA performance counters are commonly
  restricted unless the host driver allows non-admin profiling.

The job runs these commands through `run-profile.sh`:

1. `nvidia-smi`
2. `ncu --version`
3. `/workload/vectorAdd`
4. `ncu --set basic --export /tmp/nsight-compute/vectoradd ...`

The `basic` set is intentionally small enough for a PoC while still collecting
LaunchStats, Occupancy, SpeedOfLight, and WorkloadDistribution metrics. The job
also writes `/tmp/nsight-compute/vectoradd.ncu-rep` inside the container.

## Build the Image

```bash
cd test/nsight-compute-poc
make build
```

The Dockerfile starts from `nvcr.io/nvidia/cuda:13.3.0-devel-ubuntu24.04`,
matching the local host driver observed during the PoC (`nvidia-smi` reported
CUDA UMD 13.3). If `ncu` is not already present, it installs the CUDA repository
package named by `NSIGHT_COMPUTE_PACKAGE`:

```bash
docker build \
  --build-arg CUDA_IMAGE=nvcr.io/nvidia/cuda:13.3.0-devel-ubuntu24.04 \
  --build-arg NSIGHT_COMPUTE_PACKAGE=nsight-compute-2025.2.1 \
  -t kratos-nsight-compute-poc:latest .
```

For a different CUDA image version, choose the matching Nsight Compute package
from the CUDA apt repository.

If the cluster cannot download the Nsight package reliably, build from a staged
`.deb` instead:

```bash
curl -L --fail -C - \
  -o /tmp/nsight-compute-2024.1.1_2024.1.1.4-1_amd64.deb \
  https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2204/x86_64/nsight-compute-2024.1.1_2024.1.1.4-1_amd64.deb

mkdir -p /tmp/kratos-nsight-build
cp Dockerfile.offline /tmp/kratos-nsight-build/Dockerfile
cp /tmp/nsight-compute-2024.1.1_2024.1.1.4-1_amd64.deb /tmp/kratos-nsight-build/
docker build -t kratos-nsight-compute-poc:latest /tmp/kratos-nsight-build
```

## Optional Local Container Check

Requires the NVIDIA Container Toolkit on the host:

```bash
make run
```

This uses:

```bash
docker run --rm --gpus all --cap-add SYS_ADMIN kratos-nsight-compute-poc:latest ncu --version
```

Successful output should include:

- `ncu --version` output.

## Run in Kubernetes

Create or select the GPU-enabled local cluster described in
`docs/getting-started/kind-gpu.md`, then load the image if it only exists in the
local Docker daemon:

```bash
make load CLUSTER=kratos-gpu
```

Apply the job:

```bash
make apply
kubectl wait --for=condition=complete job/nsight-compute-vectoradd --timeout=180s
make logs
```

The manifest is intentionally standalone:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: nsight-compute-vectoradd
spec:
  backoffLimit: 0
  template:
    spec:
      runtimeClassName: nvidia
      restartPolicy: Never
      containers:
        - name: nsight-compute
          image: kratos-nsight-compute-poc:latest
          imagePullPolicy: IfNotPresent
          securityContext:
            capabilities:
              add:
                - SYS_ADMIN
          resources:
            limits:
              nvidia.com/gpu: 1
```

## Verification Criteria

The PoC is successful when the Job reaches `Complete` and logs show all of:

- `nvidia-smi` succeeds.
- `ncu --version` succeeds.
- the CUDA workload prints `Validation passed`.
- `ncu` prints imported raw metrics for the `vectorAdd` kernel.
- `/tmp/nsight-compute/vectoradd.ncu-rep` is listed by the script.

Validated on the local `kratos-gpu` Kind cluster:

- worker GPU capacity/allocatable: `nvidia.com/gpu=3`.
- host driver reported by `nvidia-smi`: `610.43.02`, CUDA UMD `13.3`.
- Job completed successfully with one succeeded pod.
- `ncu` collected 8 profiling passes for `vectorAdd`.
- raw imported metrics included `sm__throughput`, `lts__throughput`,
  `sm__cycles_active`, and `profiler__replayer_passes`.
- report file was generated at `/tmp/nsight-compute/vectoradd.ncu-rep`.

To copy the binary report before deleting the pod:

```bash
POD="$(kubectl get pods -l job-name=nsight-compute-vectoradd -o jsonpath='{.items[0].metadata.name}')"
kubectl cp "${POD}:/tmp/nsight-compute/vectoradd.ncu-rep" ./vectoradd.ncu-rep
```

## Known Issues and Fixes

### Profiling Counters Permission Error

Symptoms include an Nsight Compute failure mentioning insufficient privileges,
permission to access GPU performance counters, or `ERR_NVGPUCTRPERM`.

Proposed fixes:

- keep `securityContext.capabilities.add: ["SYS_ADMIN"]` for this PoC job; or
- configure the host driver to allow non-admin users to access profiling
  counters, for example with the NVIDIA `NVreg_RestrictProfilingToAdminUsers`
  module option set to `0`.

For production automation, prefer a narrow, documented cluster policy over
blanket privileged containers.

### Missing NVIDIA Runtime

If the pod cannot start with `runtimeClassName: nvidia`, confirm that the local
lab was created with `nvkind` and that Docker has NVIDIA runtime support:

```bash
docker run --rm --gpus all nvcr.io/nvidia/cuda:13.3.0-base-ubuntu24.04 nvidia-smi
kubectl get runtimeclass
```

If the cluster exposes GPUs without a runtime class, remove
`runtimeClassName: nvidia` from `job.yaml`.

### Device Plugin Does Not Advertise GPUs

If the pod is pending because `nvidia.com/gpu` is unavailable, inspect the
device plugin and node allocatable resources:

```bash
kubectl get pods -n nvidia-device-plugin
kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{" "}{.status.allocatable.nvidia\.com/gpu}{"\n"}{end}'
```

Install or fix the NVIDIA device plugin as shown in
`docs/getting-started/kind-gpu.md`.

### Driver and CUDA Compatibility

The container CUDA major version must be compatible with the host NVIDIA driver.
If `nvidia-smi` works but the workload fails at CUDA initialization, rebuild with
a CUDA image version supported by the installed driver.

### Nsight Compute Package Mismatch

If the image build cannot find the configured Nsight Compute apt package, pass a
package name that exists in the CUDA apt repository for the selected base image:

```bash
docker build \
  --build-arg CUDA_IMAGE=nvcr.io/nvidia/cuda:<matching-devel-tag> \
  --build-arg NSIGHT_COMPUTE_PACKAGE=<matching-package> \
  -t kratos-nsight-compute-poc:latest .
```

### Large Image or Package Downloads Timeout

During validation, pulling full CUDA devel images from Docker Hub/NGC stalled in
the local Docker daemon, and installing `nsight-compute-2024.1.1` from inside the
Kind pod timed out while fetching the 594 MB `.deb` from
`developer.download.nvidia.com`.

The successful workaround was:

1. use the smaller cached CUDA runtime base image,
   `nvidia/cuda:12.4.1-base-ubuntu22.04`;
2. download the Nsight Compute `.deb` on the host with resumable `curl -C -`;
3. build `Dockerfile.offline` from a temporary `/tmp` context;
4. load the resulting image with
   `kind load docker-image kratos-nsight-compute-poc:latest --name kratos-gpu`.

### Nsight Set Selection

`--set speedOfLight` is not a valid set identifier for Nsight Compute 2024.1.1.
It results in `No metrics to collect found in sections`. Use `--set basic`, or
select `SpeedOfLight` as a section explicitly.

When `--export` is used, the command may only print profiler progress. This PoC
imports the generated `.ncu-rep` with `ncu --import --page raw` before the pod
exits so metrics are visible in Kubernetes logs.

### Nsight Wrapper Path

The top-level Nsight Compute launcher depends on its real installation path to
find bundled section files. Do not symlink `/opt/nvidia/nsight-compute/.../ncu`
directly into `PATH`. `Dockerfile.offline` creates a small wrapper script in
`/usr/local/bin/ncu` that executes the installed launcher in place.
