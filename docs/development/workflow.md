# Development Workflow

Use this short loop for normal changes:

1. Put the change in the package that owns it.
2. Regenerate manifests if API or RBAC markers changed.
3. Run tests.
4. Commit related code and docs together.

## Package Ownership

- API fields and status: `api/v1alpha1`
- Reconciliation: `internal/controller`
- Profile storage: `pkg/catalog`
- Profiling and classification: `pkg/profiling`
- Scheduling constraints and scoring: `pkg/scheduling`
- Volcano resources: `pkg/volcano`
- Workload construction: `pkg/workflow`
- Metrics: `pkg/telemetry`

## Commands

```bash
make manifests
make generate
make test
```

For a manual cluster check after manifests are generated:

```bash
kubectl apply -k config/default
kubectl apply -f config/samples/gpu_v1alpha1_cudaexperiment.yaml
kubectl get cudaexperiments.gpu.scheduler.io
```

## Local Controller Smoke Test

Run the controller against the current Kubernetes context:

```bash
make install
make run
```

In another terminal, apply the sample resource:

```bash
kubectl apply -f config/samples/gpu_v1alpha1_cudaexperiment.yaml
kubectl get cudaexperiments.gpu.scheduler.io cuda-vector-add -o yaml
```
