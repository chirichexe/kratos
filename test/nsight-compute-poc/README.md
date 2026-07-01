# Nsight Compute Runner

This directory contains the profiling runner image used by the
`CUDAExperiment` controller.

The controller path is documented in
`docs/experiments/nsight-compute-poc.md`.

## Build

Default build, recommended for validation:

```bash
make build
```

This uses `Dockerfile.runtime`:

- base image: `nvidia/cuda:12.4.1-base-ubuntu22.04`;
- package: `nsight-compute-2024.1.1`;
- tools: `python3`, `kubectl`, `/scripts/profile.sh`,
  `/scripts/parse_ncu.py`.

Load into the local GPU kind cluster:

```bash
make load CLUSTER=kratos-gpu
```

## Local Smoke Check

Requires NVIDIA Container Toolkit:

```bash
docker run --rm --gpus all --cap-add SYS_ADMIN \
  --entrypoint /bin/sh kratos-nsight-compute-poc:latest \
  -lc 'nvidia-smi && ncu --version && /scripts/parse_ncu.py --help >/dev/null'
```

## How The Runner Works

The profiling runner receives:

```text
/scripts/profile.sh <workload> <report-path-without-extension> [args...]
```

It then:

1. runs `nvidia-smi`;
2. runs `ncu --version`;
3. launches the staged workload under `ncu --set basic`;
4. imports the generated `.ncu-rep` with `ncu --import --page raw`;
5. parses selected metrics into `summary.json`;
6. publishes the summary to the ConfigMap named by
   `KRATOS_PROFILE_SUMMARY_CONFIGMAP`.

The controller-created profiling Job stages the workload executable from
`spec.image` into a shared `emptyDir`, then runs this profiling runner as the
only GPU-requesting container.

## Troubleshooting

If the pod cannot access performance counters, keep
`securityContext.capabilities.add: ["SYS_ADMIN"]` for this PoC or configure the
host driver to allow non-admin profiling counters.

If the pod cannot start with `runtimeClassName: nvidia`, verify the local GPU
kind setup and runtime class:

```bash
kubectl get runtimeclass
kubectl get pods -n nvidia-device-plugin
kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{" "}{.status.allocatable.nvidia\.com/gpu}{"\n"}{end}'
```

If the image build is slow, the large part is the Nsight Compute package. Avoid
rebuilding unless `profile.sh` or `parse_ncu.py` changed; rebuild once and keep
the image loaded in kind.
