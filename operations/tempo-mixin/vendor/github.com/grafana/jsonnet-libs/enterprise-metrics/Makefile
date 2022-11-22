.ONESHELL:
.DELETE_ON_ERROR:
export SHELL     := bash
export SHELLOPTS := pipefail:errexit
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rule

# Adapted from https://suva.sh/posts/well-documented-makefiles/
.PHONY: help
help: ## Display this help.
help:
	@awk 'BEGIN {FS = ": ##"; printf "Usage:\n  make <target>\n\nTargets:\n"} /^[a-zA-Z0-9_\.\-\/%]+: ##/ { printf "  %-45s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

vendor: ## Vendor the Jsonnet libraries.
vendor: jsonnetfile.json jsonnetfile.lock.json
	jb install

test/vendor: ## Vendor the test Jsonnet libraries.
test/vendor: test/jsonnetfile.json test/jsonnetfile.lock.json
	cd test; jb install

.PHONY: test
test: ## Run library tests.
test: test/vendor
	jsonnet -J test/lib -J test/vendor -J vendor main_test.jsonnet

.PHONY: eval
eval: ## Evaluate a minimal configuration.
eval: test/vendor vendor
	jsonnet -J test/lib -J test/vendor -J vendor docs.jsonnet

environments/minio/jsonnetfile.json: ## Update the environment jsonnetfile.
environments/minio/jsonnetfile.json: environments/minio/jsonnetfile.jsonnet jsonnetfile.json test/jsonnetfile.json
	jsonnet $< > $@

environments/minio/vendor: ## Vendor environment dependencies.
environments/minio/vendor: environments/minio/jsonnetfile.json
	cd $(@D); jb install

.PHONY: local-test
local-test: ## Deploy GEM to a local k3d cluster. Tear down with ./scripts/k3d-cluster delete.
local-test: environments/minio/vendor
	./scripts/k3d-cluster create
	tk apply --tla-code kubeconfig="$$(kubectl config view -o json)" environments/minio

.PHONY: docs
docs: ## Remake the library README file.
docs: docs/README.md

docs/README.md: main.libsonnet vendor test/vendor
	docsonnet -J test/lib -J test/vendor -J vendor docs.jsonnet -o ./docs
