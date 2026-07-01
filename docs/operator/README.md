# Operator

`CUDAExperiment` is the user-facing resource. Users describe the workload
container, command, GPU request, runtime class, and whether KRATOS should profile
before normal execution.

## Current Reconciliation Flow

When `spec.profilingEnabled` is false:

1. ensure `<experiment>-execution` exists;
2. record `ExecutionJobCreated` in status.

When `spec.profilingEnabled` is true:

1. check for `WorkloadProfile` named `<experiment>-profile`;
2. if the profile is missing, ensure `<experiment>-profiling` exists;
3. when the profiling Job completes, read
   `<experiment>-profile-summary` ConfigMap;
4. create/update `<experiment>-profile` and write parsed metrics into
   `WorkloadProfile.status`;
5. on a later reconcile, observe the profile and ensure
   `<experiment>-execution` exists.

The profiling and execution Jobs are mutually exclusive until a profile exists.

## Profiling Job

The profiling Job keeps the workload image independent from Nsight Compute:

- `stage-workload` initContainer uses `spec.image`, verifies
  `spec.command[0]`, copies the CUDA executable into a shared `emptyDir`, and
  exits.
- `profiling-runner` uses `kratos-nsight-compute-poc:latest` by default,
  requests `nvidia.com/gpu`, launches the staged executable under
  `ncu --set basic`, imports raw metrics, parses a summary, and publishes a
  ConfigMap.

The runner image can be overridden with `KRATOS_NSIGHT_COMPUTE_IMAGE` on the
controller manager.

For the NVIDIA sample container use:

```yaml
spec:
  image: nvcr.io/nvidia/k8s/cuda-sample:vectoradd-cuda12.5.0
  command:
    - /cuda-samples/vectorAdd
  runtimeClassName: nvidia
  gpuRequired: 1
  replicas: 1
  profilingEnabled: true
```

## Outputs

Profiling writes:

- Job: `<experiment>-profiling`
- ConfigMap: `<experiment>-profile-summary`
- WorkloadProfile: `<experiment>-profile`

Example `WorkloadProfile.status` from a real Nsight run:

```yaml
status:
  boundType: unknown
  metrics:
    achievedOccupancy: "78.21"
    l2Throughput: "30.22"
    smThroughput: "12.56"
```

Inspect:

```bash
kubectl logs job/<experiment>-profiling
kubectl get configmap <experiment>-profile-summary -o yaml
kubectl get workloadprofile <experiment>-profile -o yaml
kubectl get cudaexperiment <experiment> -o yaml
```

## Current Caveat

The controller watches `CUDAExperiment` and owned Jobs. It does not yet watch
owned `WorkloadProfile` resources. After the controller creates a profile,
another reconcile is needed before the execution Job is created. For manual
validation:

```bash
kubectl annotate cudaexperiment <experiment> \
  validation.gpu.scheduler.io/reconcile-at=manual --overwrite
```

The next controller step should add a `WorkloadProfile` watch or explicitly
requeue after successful profile creation.

## Longer-Term Direction

The current controller only validates the profiling pipeline. Future scheduling
work should use workload profiles, GPU telemetry, node constraints, and Volcano
integration to produce placement hints before execution.
