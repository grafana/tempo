.ONESHELL:
.DELETE_ON_ERROR:

export SHELL     := bash
export SHELLOPTS := pipefail:errexit

MAKEFLAGS += --warn-undefined-variables --no-builtin-rule
K8S_VERSION := 1.18

# Adapted from https://suva.sh/posts/well-documented-makefiles/
.PHONY: help
help: ## Display this help.
help:
	@awk 'BEGIN {FS = ": ##"; printf "Usage:\n  make <target>\n\nTargets:\n"} /^[a-zA-Z0-9_\.\-\/% ]+: ##/ { printf "  %-45s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

jsonnetfile.lock.json: ## Update the locked dependency versions for the library.
jsonnetfile.lock.json: jsonnetfile.json
	jb update

test: ## Evaluate the library Jsonnet.
test: test/.eval
	:

test/.eval: # Cache testing results.
test/.eval: test/eval.jsonnet main.libsonnet test/lib/k.libsonnet test/jsonnetfile.lock.json test/vendor
	cd $(@D); tmp=$$(mktemp); if tk eval $(<F) | tee $${tmp}; then mv $${tmp} $(@F); fi

test/lib:
	mkdir -p $@

test/lib/k.libsonnet: # Install implicit k.libsonnet dependency used in testing.
test/lib/k.libsonnet: test/lib
	printf "(import 'github.com/jsonnet-libs/k8s-libsonnet/1.18/main.libsonnet')" > $@

test/jsonnetfile.lock.json test/vendor: ## Update the locked dependency versions used in testing.
test/jsonnetfile.lock.json test/vendor: test/jsonnetfile.json
	cd $(@D); jb update
