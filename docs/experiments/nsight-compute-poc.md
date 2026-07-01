# Nsight Compute Profiling PoC

This PoC validates the current profiling pipeline end to end:

1. a `CUDAExperiment` with `spec.profilingEnabled: true` creates a profiling
   Job;
2. the profiling Job stages the workload executable from the workload image;
3. the profiling runner launches that executable under Nsight Compute;
4. the runner publishes a profile summary ConfigMap;
5. the controller creates a `WorkloadProfile` and writes parsed metrics into
   `status`;
6. a later reconcile observes the profile and creates the execution Job.

The parser and classification are still intentionally small. The important PoC
signal is that the metrics come from a real `ncu` run, not from synthetic data.

## Images

Build the controller:

```bash
make docker-build IMG=kratos-controller:profiling-poc
kind load docker-image kratos-controller:profiling-poc --name kratos-gpu
make deploy IMG=kratos-controller:profiling-poc
kubectl rollout restart -n kratos-system deployment/kratos-controller-manager
kubectl rollout status -n kratos-system deployment/kratos-controller-manager
```

Build the profiling runner:

```bash
cd test/nsight-compute-poc
make build
make load CLUSTER=kratos-gpu
```

`make build` uses `Dockerfile.runtime`, based on
`nvidia/cuda:12.4.1-base-ubuntu22.04`, and installs only the runtime pieces
needed by the PoC: Nsight Compute CLI, Python, `kubectl`, the parser, and
`/scripts/profile.sh`.

## Test Experiment

Apply the CRDs/RBAC if they are not already deployed:

```bash
kubectl apply -f config/crd/bases/gpu.scheduler.io_workloadprofiles.yaml
kubectl apply -f config/rbac/profiling_runner_configmap_role.yaml
kubectl apply -f config/rbac/profiling_runner_configmap_role_binding.yaml
```

Create or reapply a profiling experiment:

```yaml
apiVersion: gpu.scheduler.io/v1alpha1
kind: CUDAExperiment
metadata:
  name: cuda-vector-add-validation
  namespace: default
spec:
  image: nvcr.io/nvidia/k8s/cuda-sample:vectoradd-cuda12.5.0
  command:
    - /cuda-samples/vectorAdd
  runtimeClassName: nvidia
  replicas: 1
  gpuRequired: 1
  profilingEnabled: true
```

For a clean rerun:

```bash
kubectl delete workloadprofile cuda-vector-add-validation-profile --ignore-not-found
kubectl delete configmap cuda-vector-add-validation-profile-summary --ignore-not-found
kubectl delete job cuda-vector-add-validation-profiling --ignore-not-found
kubectl delete job cuda-vector-add-validation-execution --ignore-not-found
kubectl apply -f tmp/cudaexperiment-profiling-validation.yaml
```

Check the pipeline:

```bash
kubectl get jobs
kubectl get pods
kubectl logs job/cuda-vector-add-validation-profiling
kubectl get configmap cuda-vector-add-validation-profile-summary -o yaml
kubectl get workloadprofile cuda-vector-add-validation-profile -o yaml
kubectl get cudaexperiment cuda-vector-add-validation -o yaml
```

Expected markers:

- profiling Job reaches `Complete`;
- logs include `ncu --version`, `Profiling "vectorAdd"`, and `Test PASSED`;
- logs include real raw metrics such as `sm__throughput` and
  `lts__throughput`;
- ConfigMap `<experiment>-profile-summary` contains `summary.json`;
- `WorkloadProfile.status.boundType` and `WorkloadProfile.status.metrics` are
  populated.

Example result from the local `kratos-gpu` cluster:

```yaml
status:
  boundType: unknown
  metrics:
    achievedOccupancy: "78.21"
    l2Throughput: "30.22"
    smThroughput: "12.56"
```

`unknown` is acceptable for the NVIDIA `vectorAdd` sample with the current
thresholds. The PoC goal is real metric capture and controller propagation.

## Current Caveats

- The controller currently watches `CUDAExperiment` and owned Jobs. It does not
  watch `WorkloadProfile` yet, so after a profile is created a later reconcile
  is needed to create the execution Job. For manual validation:

  ```bash
  kubectl annotate cudaexperiment cuda-vector-add-validation \
    validation.gpu.scheduler.io/reconcile-at=manual --overwrite
  ```

- The runner needs GPU performance counter access. The PoC grants `SYS_ADMIN`
  to the profiling container. A production setup should replace that with an
  explicit cluster security policy.
- The Nsight package is large, around 594 MB. Keep the built image loaded in
  kind during validation.
- `ncu --set basic` is used because it works with Nsight Compute 2024.1.1 and
  collects enough metrics for this phase.
