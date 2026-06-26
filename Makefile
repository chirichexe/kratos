SHELL := /usr/bin/env bash

.PHONY: help
help:
	@echo "KRATOS development targets"
	@echo "  make manifests         Generate CRDs and RBAC manifests"
	@echo "  make generate          Generate deepcopy code"
	@echo "  make test              Run Go tests"

.PHONY: manifests
manifests:
	controller-gen rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate:
	controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: test
test:
	go test ./...
