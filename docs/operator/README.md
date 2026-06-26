# Operator

`CUDAExperiment` is the user-facing resource. Users describe the container,
GPU requirements, profiling preference, and optional distributed constraints.

Expected lifecycle:

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
