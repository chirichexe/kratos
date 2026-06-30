# Operator

`CUDAExperiment` is the user-facing resource. Users describe the container,
GPU requirements, profiling preference, and optional distributed constraints.

Current minimal lifecycle:

1. Read the `CUDAExperiment` spec.
2. Create a Kubernetes Job named `<experiment-name>-execution`.
3. Set the Job pod template image, command, arguments, GPU limit, and runtime
   class from the experiment spec.
4. Set an owner reference from the Job to the `CUDAExperiment`.
5. Record the Job name and `ExecutionJobCreated` condition in status.

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
