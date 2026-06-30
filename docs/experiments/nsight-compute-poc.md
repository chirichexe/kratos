# Nsight Compute Kubernetes PoC

The current Nsight Compute work is a feasibility proof, not controller
automation. The runnable artifacts live in `test/nsight-compute-poc`.

## What It Demonstrates

The PoC launches a Kubernetes `Job` that:

1. requests one GPU with `nvidia.com/gpu: 1`;
2. uses `runtimeClassName: nvidia` for the local `nvkind` GPU lab;
3. runs `nvidia-smi`;
4. runs `ncu --version`;
5. profiles a small CUDA `vectorAdd` kernel with `ncu`;
6. writes an Nsight Compute report at
   `/tmp/nsight-compute/vectoradd.ncu-rep`.

This deliberately ignores `CUDAExperiment`, automatic controller logic, Python
parsing, and workload classification.

## Runbook

Build and load the image:

```bash
cd test/nsight-compute-poc
make build
make load CLUSTER=kratos-gpu
```

Run the Kubernetes job:

```bash
make apply
kubectl wait --for=condition=complete job/nsight-compute-vectoradd --timeout=180s
make logs
```

The logs should include `nvidia-smi`, `ncu --version`, `Validation passed`, and
Nsight Compute metric output for the `vectorAdd` kernel.

For full details, including image build arguments and troubleshooting, see
`test/nsight-compute-poc/README.md`.

## Expected Cluster Requirements

- NVIDIA host driver compatible with the selected CUDA image.
- Docker configured with the NVIDIA Container Toolkit.
- A GPU-capable Kubernetes cluster, such as the local `nvkind` lab in
  `docs/getting-started/kind-gpu.md`.
- NVIDIA device plugin advertising `nvidia.com/gpu`.
- Permission for Nsight Compute to access GPU performance counters.

## Known Operational Caveats

Nsight Compute may fail with a profiling-counter permission error unless the pod
has `SYS_ADMIN` capability or the host driver allows non-admin profiling
counters. This PoC adds `SYS_ADMIN` to the container manifest to validate
feasibility. A production implementation should replace that with a deliberate
cluster policy.

If the job remains pending, verify that the NVIDIA device plugin advertises
allocatable GPUs. If the pod cannot start with `runtimeClassName: nvidia`, verify
the local runtime class or remove that field on clusters that expose GPUs
without a runtime class.

## Validation Result

The PoC was validated on the local `kratos-gpu` Kind cluster. The worker node
advertised three logical GPU slots through `nvidia.com/gpu`, the Job requested
one GPU, and the pod completed successfully.

The successful run showed:

- `nvidia-smi` inside the pod with driver `610.43.02` and CUDA UMD `13.3`;
- `ncu --version` reporting Nsight Compute `2024.1.1`;
- NVIDIA `vectorAdd` sample output ending in `Test PASSED`;
- Nsight Compute profiling `vectorAdd` in 8 passes;
- raw imported metrics including `sm__throughput`, `lts__throughput`,
  `sm__cycles_active`, and `profiler__replayer_passes`;
- `/tmp/nsight-compute/vectoradd.ncu-rep` generated in the pod.

The main issues encountered were large CUDA image pulls stalling, in-cluster
download timeout for the 594 MB Nsight Compute package, and an invalid initial
`--set speedOfLight` option. The working path uses `Dockerfile.offline`, a
host-downloaded Nsight Compute `.deb`, `--set basic`, and an explicit
`ncu --import --page raw` step so metrics are visible in Kubernetes logs.
