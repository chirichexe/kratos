# Development Workflow

Use this short loop for normal changes:

1. Put the change in the package that owns it.
2. Regenerate manifests if API or RBAC markers changed.
3. Run tests.
4. Commit related code and docs together.

## Package Ownership

- API fields and status: `api/v1alpha1`
- Reconciliation: `internal/controller`
- Profiling: `pkg/profiling`
- Profile storage: `pkg/catalog`
- Scheduling: `pkg/scheduling`
- Argo resources: `pkg/workflow`
- Volcano resources: `pkg/volcano`
- Metrics: `pkg/telemetry`

## Commands

```bash
make manifests
make generate
go test ./...
```

For a manual cluster check after manifests are generated:

```bash
kubectl apply -k config/default
kubectl apply -f config/samples/kratos_v1alpha1_aiexperiment.yaml
kubectl get aiexperiments.kratos.io
```
