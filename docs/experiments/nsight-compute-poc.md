# Nsight Compute Profiling PoC

This PoC validates the current profiling pipeline end to end:

1. A `CUDAExperiment` with `spec.profilingEnabled: true` creates a profiling Job;
2. The profiling Job stages the workload executable from the workload image;
3. The profiling runner launches that executable under Nsight Compute (`ncu`);
4. The runner publishes a profile summary ConfigMap;
5. The controller creates a `WorkloadProfile` and writes parsed metrics into `status`;
6. Upon profiling Job completion, reconciliation automatically creates the execution Job.

The parser and classification are intentionally small. The key PoC signal is that metrics come from a real `ncu` run on hardware, not from synthetic data.

## Images

Build and deploy the controller:

```bash
make docker-build IMG=kratos-controller:v0.1.0
kind load docker-image kratos-controller:v0.1.0 --name kratos-gpu
make deploy IMG=kratos-controller:v0.1.0
kubectl rollout status -n kratos-system deployment/kratos-controller-manager
```

Build and load the profiling runner:

```bash
cd test/nsight-compute-poc
make build
make load CLUSTER=kratos-gpu
cd ../..
```

`make build` uses `Dockerfile.runtime` (based on `nvidia/cuda:12.4.1-base-ubuntu22.04`) and installs Nsight Compute CLI, Python, `kubectl`, the metric parser, and `/scripts/profile.sh`.

## Test Experiment

Apply the CRDs and RBAC permissions if they are not already deployed:

```bash
make install
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
kubectl apply -f config/samples/gpu_v1alpha1_cudaexperiment.yaml
```

Check the pipeline execution:

```bash
kubectl get jobs
kubectl get pods
kubectl logs job/cuda-vector-add-validation-profiling
kubectl get configmap cuda-vector-add-validation-profile-summary -o yaml
kubectl get workloadprofile cuda-vector-add-validation-profile -o yaml
kubectl get cudaexperiment cuda-vector-add-validation -o yaml
kubectl logs job/cuda-vector-add-validation-execution
```

Expected indicators:

- Profiling Job reaches `Complete`;
- Logs include `ncu --version`, `Profiling "vectorAdd"`, and `Test PASSED`;
- Logs include real raw metrics such as `sm__throughput` and `lts__throughput`;
- ConfigMap `<experiment>-profile-summary` contains `summary.json`;
- `WorkloadProfile.status.boundType` and `WorkloadProfile.status.metrics` are populated;
- Execution Job `<experiment>-execution` is created and reaches `Complete`.

Example result from the local `kratos-gpu` cluster:

```yaml
status:
  boundType: unknown
  metrics:
    achievedOccupancy: "80.62"
    l2Throughput: "31.23"
    smThroughput: "12.58"
```

`unknown` is expected for the NVIDIA `vectorAdd` sample with the current thresholds. The PoC goal is real metric capture and controller propagation.

## Notes & Security Constraints

- The runner container requests GPU performance counter access. The PoC grants `SYS_ADMIN` capability to the profiling container.
- The Nsight package image is around 594 MB. Keep the image loaded in Kind during validation.
- `ncu --set basic` is used because it works with Nsight Compute 2024.1.1 and collects essential hardware metrics.
