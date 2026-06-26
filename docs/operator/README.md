# Operator

`CudaExperiment` is the user-facing resource. Users describe the model,
dataset, batch size, epochs, priority, and precision; the controller decides
how to run it.

Expected lifecycle:

1. Read the `CudaExperiment` spec.
2. Look up an existing workload profile.
3. Run a reduced profiling job when no profile exists.
4. Classify the workload.
5. Select a Volcano queue and GPU or MIG shape.
6. Create or update Argo and Volcano resources.
7. Update status and expose telemetry.

The reconciler should coordinate this flow, while reusable decisions stay in
`pkg/catalog`, `pkg/profiling`, `pkg/scheduling`, `pkg/workflow`,
`pkg/volcano`, and `pkg/telemetry`.

Generated resources should be owned by the `CudaExperiment` so Kubernetes garbage
collection can clean them up.
