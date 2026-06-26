# Development Workflow

Use this workflow for normal project changes.

## 1. Create or Update the Design

Before writing controller logic, identify which part of the system owns the
change:

- API field or status change: `api/v1alpha1`
- Reconciliation behavior: `internal/controller`
- Profiling behavior: `pkg/profiling`
- Profile persistence: `pkg/catalog`
- Scheduling policy: `pkg/scheduling`
- Argo resource construction: `pkg/workflow`
- Volcano resource construction: `pkg/volcano`
- Metrics export: `pkg/telemetry`

Update documentation when the workflow or module responsibility changes.

## 2. Implement the Change

Keep controller code thin. The reconciler should orchestrate actions and call
domain packages for catalog lookup, profiling, scheduling, workflow generation,
Volcano object generation, and telemetry.

Prefer small packages with explicit inputs and outputs. This makes scheduling
policies and profiling logic easier to test without a Kubernetes cluster.

## 3. Generate Kubernetes Assets

After API or RBAC marker changes, regenerate manifests:

```bash
make manifests
make generate
```

Generated CRDs belong in `config/crd/bases`.

## 4. Verify Locally

Run the Go checks:

```bash
go test ./...
```

When the real controller is implemented, add focused unit tests for domain
packages and reconciler tests for Kubernetes object ownership and status
updates.

## 5. Test in a Cluster

For manual testing, apply the default kustomize entrypoint after manifests are
generated:

```bash
kubectl apply -k config/default
kubectl apply -f config/samples/kratos_v1alpha1_aiexperiment.yaml
```

Inspect the custom resource status and generated workloads:

```bash
kubectl get aiexperiments.kratos.io
kubectl describe aiexperiment resnet50-cifar100
```

## 6. Commit and Push

Before committing:

```bash
git status --short
go test ./...
```

Commit related code and documentation together so the repository remains
understandable at every revision.
