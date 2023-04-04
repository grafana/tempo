.ONESHELL:
.DELETE_ON_ERROR:
export SHELL     := bash
export SHELLOPTS := pipefail:errexit
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rule

.DEFAULT_GOAL: help

# Adapted from https://www.thapaliya.com/en/writings/well-documented-makefiles/
.PHONY: help
help: ## Display this help.
help:
	@awk 'BEGIN {FS = ": ##"; printf "Usage:\n  make <target>\n\nTargets:\n"} /^[a-zA-Z0-9_\.\-\/%]+: ##/ { printf "  %-45s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

GIT_ROOT := $(shell git rev-parse --show-toplevel)

# List of projects to provide to the make-docs script.
PROJECTS := tempo loki

# Name for the container.
export DOCS_CONTAINER := $(firstword $(subst /,-,$(PROJECTS))-docs)

# Host port to publish container port to.
export DOCS_HOST_PORT := 3002

# Container image used to perform Hugo build.
export DOCS_IMAGE := grafana/docs-base:latest

# PATH-like list of directories within which to find projects.
# If all projects are checked out into the same directory, ~/repos/ for example, then the default should work.
export REPOS_PATH := $(realpath $(GIT_ROOT)/..)

# How to treat Hugo relref errors.
export HUGO_REFLINKSERRORLEVEL := WARNING

.PHONY: docs-rm
docs-rm: ## Remove the docs container.
	$(PODMAN) rm -f $(DOCS_CONTAINER)

.PHONY: docs-pull
docs-pull: ## Pull documentation base image.
	$(PODMAN) pull $(DOCS_IMAGE)

make-docs: ## Fetch the latest make-docs script.
make-docs:
	curl -s -LO https://raw.githubusercontent.com/grafana/writers-toolkit/main/scripts/make-docs
	chmod +x make-docs

.PHONY: docs
docs: ## Serve documentation locally.
docs: make-docs
	$(PWD)/make-docs $(PROJECTS)

.PHONY: docs/lint
docs/lint: ## Run docs-validator on the entire docs folder.
	$(PODMAN) run --rm -ti \
		--platform linux/amd64 \
		--volume "$(GIT_ROOT)/docs/sources:/docs/sources" \
		grafana/doc-validator:latest \
		--skip-image-validation \
		/docs/sources \
		/docs/k6

.PHONY: doc-validator/%
doc-validator/%: ## Run doc-validator on a specific path. To lint the path /docs/sources/administration, run 'make doc-validator/administration'.
doc-validator/%:
	$(PODMAN) run --init -v "$(shell pwd)/sources:/sources" grafana/doc-validator:latest --skip-image-validation --include=$(subst doc-validator/,,$@) ./sources /docs/grafana/latest
