# Operator

`CUDAExperiment` is the user-facing resource. Users describe the container,
GPU requirements, profiling preference, and optional distributed constraints.

Current minimal lifecycle:

1. Read the `CUDAExperiment` spec.
2. Create a Kubernetes Job named `<experiment-name>-execution`.
3. Set the Job pod template image, command, arguments, GPU limit, profiling
   runner, and runtime class from the experiment spec.
4. Set an owner reference from the Job to the `CUDAExperiment`.
5. Record the Job name and `ExecutionJobCreated` condition in status.

When `spec.profilingEnabled` is false, the Job contains the original single
`execution` container that runs the configured image, command, arguments, and
GPU limit.

When `spec.profilingEnabled` is true, the Job contains:

- `workload`: the experiment image. It copies the CUDA executable into a shared
  `emptyDir` volume and waits for profiling to complete.
- `nsight-compute-sidecar`: the controller-owned Nsight Compute profiling
  runner. It requests `nvidia.com/gpu`, runs `nvidia-smi`, runs `ncu --version`,
  launches the staged workload under `ncu --set basic`, imports the `.ncu-rep`,
  and prints raw metrics to its logs.

The profiling runner image defaults to `kratos-nsight-compute-poc:latest`.
Override it by setting `KRATOS_NSIGHT_COMPUTE_IMAGE` on the controller manager.

For custom workload images, set `spec.command[0]` to the CUDA executable path
inside the workload image. If no command is provided, the controller defaults to
`/cuda-samples/vectorAdd`, which supports the NVIDIA test container
`nvcr.io/nvidia/k8s/cuda-sample:vectoradd-cuda12.5.0`.

Inspect profiling output with:

```bash
kubectl logs job/<experiment-name>-execution -c nsight-compute-sidecar
```

The local GPU Kind setup requires `runtimeClassName: nvidia` so CUDA pods are
started through the NVIDIA runtime handler. The CRD defaults this field to
`nvidia`, and users can override it when running on clusters with a different
runtime class name.

The controller keeps the `CUDAExperiment` and completed Job after execution so
users can inspect status, events, and logs. To rerun the same experiment name,
delete the generated Job or create a new `CUDAExperiment` with a different
name.

Expected long-term lifecycle:

1. Read the `CUDAExperiment` spec.
2. Compute or read the workload hash.
3. Look up a matching profile in the knowledge base.
4. Collect static GPU data and runtime node metrics.
5. Filter nodes that do not satisfy hard constraints.
6. Score the remaining nodes.
7. Generate `NodeAffinity` and `NodeSelector` hints.
8. Submit the workload to Volcano for final scheduling.
9. Profile completed workloads and update the profile.

The reconciler should coordinate this flow while reusable decisions stay in
`pkg/catalog`, `pkg/profiling`, `pkg/scheduling`, `pkg/volcano`,
`pkg/workflow`, and `pkg/telemetry`.
