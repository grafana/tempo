.ONESHELL:
.DELETE_ON_ERROR:
SHELL       := bash
SHELLOPTS   := -euf -o pipefail
MAKEFLAGS   += --warn-undefined-variables
MAKEFLAGS   += --no-builtin-rule

# Adapted from https://suva.sh/posts/well-documented-makefiles/
.PHONY: help
help: ## Display this help
help:
	@awk 'BEGIN {FS = ": ##"; printf "Usage:\n  make <target>\n\nTargets:\n"} /^[a-zA-Z0-9_\.\-\/%]+: ##/ { printf "  %-45s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

JSONNET_FILES := $(shell find . -name 'vendor' -prune -o -name '*.jsonnet' -print -o -name '*.libsonnet' -print)
JSONNET := jsonnet -J vendor

.PHONY: fmt
fmt: ## Format all files
fmt:
	for file in $(JSONNET_FILES); do jsonnetfmt -i "$${file}"; done

.PHONY: lint
lint: ## Lint mixin
lint:
	mixtool lint mixin.libsonnet

build: ## Build rules and dashboards
build: alerts/general.yaml rules/rules.yaml dashboards/haproxy-overview.json dashboards/haproxy-backend.json dashboards/haproxy-frontend.json dashboards/haproxy-server.json

alerts/general.yaml: ## Export the general alert rules as YAML
alerts/general.yaml: alerts/general.jsonnet
	$(MAKE) fmt JSONNET_FILES="$<"
	$(JSONNET) $< > $@

rules/rules.yaml: ## Export recording rules rules as YAML
rules/rules.yaml: rules/rules.jsonnet
	$(MAKE) fmt JSONNET_FILES="$<"
	$(JSONNET) $< > $@

.PHONY: haproxy-mixin-build-image
haproxy-mixin-build-image: ## Build the haproxy-mixin-build-image
haproxy-mixin-build-image: build-image.nix common.nix $(wildcard nix/*nix)
	nix build .#haproxy-mixin-build-image -o $@
	if ! docker load --input $@; then rm $@; fi

.PHONY: inspect-build-image
inspect-build-image: ## Inspect the haproxy-mixin-build-image
inspect-build-image:
	docker save jdbgrafana/haproxy-mixin-build-image | tar x --to-stdout --wildcards '*/layer.tar' | tar tv | sort -nr -k3

dist:
	mkdir -p dist

dist/haproxy-mixin.tar.gz: ## Create a release of the haproxy-mixin artifacts
dist/haproxy-mixin.tar.gz: $(wildcard dashboards/*.json) $(wildcard alerts/*yaml) $(wildcard rules/*.yaml) $(wildcard img/*.png) | dist
	tar -cz -f $@ $^
