# KRATOS Documentation

This directory explains how to work with KRATOS as the project grows from the
current Kubebuilder-oriented scaffold into a complete Kubernetes controller.

## Reading Path

Start here:

1. [Getting Started](getting-started/README.md): local tools, repository setup,
   and basic verification commands.
2. [Project Layout](development/project-layout.md): where each module belongs
   and how code should be organized.
3. [Development Workflow](development/workflow.md): how to make changes,
   generate manifests, test, and submit work.
4. [Operator Guide](operator/README.md): how the `AIExperiment` controller is
   expected to work.
5. [Experiment Guide](experiments/README.md): how validation scenarios and
   metrics should be organized.

## Documentation Rules

- Keep documentation in English.
- Prefer short pages with a clear purpose over long mixed guides.
- Update docs in the same change that introduces or changes a workflow.
- Use repository-relative paths and runnable commands when possible.
