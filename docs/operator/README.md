# Operator Guide

The KRATOS operator exposes `AIExperiment` as the user-facing abstraction for
deep learning workloads. Users should not need to describe Pods, GPU requests,
Volcano Jobs, Argo Workflows, or MLflow setup directly.

## AIExperiment Lifecycle

1. A user creates an `AIExperiment`.
2. The controller reads the desired model, dataset, batch size, epochs,
   priority, and precision.
3. The controller checks the workload profiling catalog.
4. If no profile exists, the controller starts a reduced profiling workflow.
5. Profiling results are normalized and stored.
6. The workload is classified as `compute-bound`, `memory-bound`, or
   `balanced`.
7. A scheduling decision selects the Volcano queue, GPU shape, and MIG profile.
8. The controller creates or updates the Argo Workflow and Volcano Job.
9. The controller updates `AIExperiment` status and exposes telemetry.

## Reconciler Responsibilities

The reconciler in `internal/controller` should coordinate the lifecycle, but it
should not contain all business logic directly. It should call:

- `pkg/catalog` for profile lookup and persistence.
- `pkg/profiling` for profiling job planning and workload classification.
- `pkg/scheduling` for queue, GPU, MIG, and co-location decisions.
- `pkg/workflow` for Argo Workflow construction.
- `pkg/volcano` for Volcano Job construction.
- `pkg/telemetry` for Prometheus metric export.

## Status Fields

`AIExperimentStatus` should report enough state for users and tests to
understand progress:

- current phase
- selected workload class
- profile name or key
- generated workflow name
- generated Volcano Job name
- relevant error conditions

## Ownership Rules

Generated resources should be owned by the `AIExperiment` where possible. This
lets Kubernetes garbage collection remove workflows and jobs when the experiment
is deleted.
