# Adapted from https://www.thapaliya.com/en/writings/well-documented-makefiles/
.PHONY: help
help:  ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[[:alnum:]_-]+:.*##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.DEFAULT_GOAL:=help

# Version number, read from the VERSION file at the repo root
VERSION=$(shell ./tools/version-tag.sh)

GIT_REVISION := $(shell git rev-parse --short HEAD)
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD)

GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

GOPATH := $(shell go env GOPATH)
GORELEASER := $(GOPATH)/bin/goreleaser

# Build Images
LOKI_BUILD_IMAGE ?= grafana/loki-build-image:0.35.0
# https://hub.docker.com/repository/docker/grafana/tempo-ci-tools/
# built by: .github/workflows/docker-ci-tools.yml
TEMPO_CI_TOOLS_IMAGE ?= grafana/tempo-ci-tools:main-b0f57ed-20260527-203411
DOCS_IMAGE ?= grafana/docs-base:latest

# More exclusions can be added similar with: -not -path './testbed/*'
ALL_SRC := $(shell find . -name '*.go' \
								-not -path './tools*/*' \
								-not -path './vendor*/*' \
								-not -path './integration/*' \
                                -type f | sort)

# ALL_SRC but without pkg and tempodb packages
OTHERS_SRC := $(shell find . -name '*.go' \
								-not -path './tools*/*' \
								-not -path './vendor*/*' \
								-not -path './integration/*' \
								-not -path './pkg*/*' \
								-not -path './tempodb*/*' \
                                -type f | sort)

# All source code and documents. Used in spell check.
ALL_DOC := $(shell find . \( -name "*.md" -o -name "*.yaml" \) \
                                -type f | sort)

# ALL_PKGS is used with 'go cover'
ALL_PKGS := $(shell go list $(sort $(dir $(ALL_SRC))))

LD_FLAGS=-X main.Branch=$(GIT_BRANCH) -X main.Revision=$(GIT_REVISION) -X main.Version=$(VERSION)
ifeq ($(BUILD_DEBUG),)
	LD_FLAGS+= -w
endif

GO_OPT= -mod vendor -ldflags "$(LD_FLAGS)"
ifeq ($(BUILD_DEBUG), 1)
	GO_OPT+= -gcflags="all=-N -l"
endif

GO_ENV=CGO_ENABLED=0
ifeq ($(GOARCH),amd64)
	GO_ENV+= GOAMD64=v2
endif
ifeq ($(GOARCH),arm64)
	GO_ENV+= GOARM64=v8.0
endif

GOTEST_OPT?= -race -timeout 25m -count=1 -v
GOTEST_OPT_WITH_COVERAGE = $(GOTEST_OPT) -cover
GOTEST=gotestsum --format=testname --
COVERAGE_DIR?= .coverage
LINT=golangci-lint

UNAME := $(shell uname -s)
ifeq ($(UNAME), Darwin)
    SED_OPTS := ''
endif

FILES_TO_FMT=$(shell find . -type d \( -path ./vendor -o -path ./tools/vendor -o -path ./opentelemetry-proto -o -path ./vendor-fix -o -path ./.claude \) -prune -o -name '*.go' -not -name "*.pb.go" -not -name '*.y.go' -not -name '*.gen.go' -print)
FILES_TO_JSONNETFMT=$(shell find ./operations/jsonnet ./operations/tempo-mixin ./example -type f \( -name '*.libsonnet' -o -name '*.jsonnet' \) -not -path "*/vendor/*" -print)

##@ Building
.PHONY: tempo 	
tempo: ## Build tempo
	$(GO_ENV) go build $(GO_OPT) -o ./bin/$(GOOS)/tempo-$(GOARCH) $(BUILD_INFO) ./cmd/tempo

.PHONY: tempo-query
tempo-query: ## Build tempo-query
	$(GO_ENV) go build $(GO_OPT) -o ./bin/$(GOOS)/tempo-query-$(GOARCH) $(BUILD_INFO) ./cmd/tempo-query

.PHONY: tempo-cli
tempo-cli: ## Build tempo-cli
	$(GO_ENV) go build $(GO_OPT) -o ./bin/$(GOOS)/tempo-cli-$(GOARCH) $(BUILD_INFO) ./cmd/tempo-cli

.PHONY: tempo-vulture  ## Build tempo-vulture
tempo-vulture:
	$(GO_ENV) go build $(GO_OPT) -o ./bin/$(GOOS)/tempo-vulture-$(GOARCH) $(BUILD_INFO) ./cmd/tempo-vulture

.PHONY: exe  ## Build exe
exe:
	GOOS=linux make $(COMPONENT)

.PHONY: exe-debug  ## Build exe-debug
exe-debug:
	BUILD_DEBUG=1 GOOS=linux make $(COMPONENT)

##@ Unit Tests

.PHONY: test
test: ## Run tests
	$(GOTEST) $(GOTEST_OPT) $(ALL_PKGS)

.PHONY: benchmark
benchmark: tools ## Run benchmarks
	$(GOTEST) -bench=. -run=notests $(ALL_PKGS)

# Not used in CI, tests are split in pkg, tempodb, tempodb-wal and others in CI jobs
.PHONY: test-with-cover
test-with-cover: tools ## Run tests with code coverage
	mkdir -p $(COVERAGE_DIR)
	$(GOTEST) $(GOTEST_OPT) -coverprofile=$(COVERAGE_DIR)/all.out $(ALL_PKGS)

# tests in pkg
.PHONY: test-with-cover-pkg
test-with-cover-pkg: tools  ## Run Tempo packages' tests with code coverage
	mkdir -p $(COVERAGE_DIR)
	$(GOTEST) $(GOTEST_OPT) -coverprofile=$(COVERAGE_DIR)/pkg.out $(shell go list $(sort $(dir $(shell find . -name '*.go' -path './pkg*/*' -type f | sort))))

# tests in tempodb (excluding tempodb/wal)
.PHONY: test-with-cover-tempodb
test-with-cover-tempodb: tools ## Run tempodb tests with code coverage
	mkdir -p $(COVERAGE_DIR)
	GOMEMLIMIT=6GiB $(GOTEST) $(GOTEST_OPT) -coverprofile=$(COVERAGE_DIR)/tempodb.out $(shell go list $(sort $(dir $(shell find . -name '*.go'  -not -path './tempodb/wal*/*' -path './tempodb*/*' -type f | sort))))

# tests in tempodb/wal
.PHONY: test-with-cover-tempodb-wal
test-with-cover-tempodb-wal: tools  ## Test tempodb/wal with code coverage
	mkdir -p $(COVERAGE_DIR)
	$(GOTEST) $(GOTEST_OPT) -coverprofile=$(COVERAGE_DIR)/tempodb-wal.out $(shell go list $(sort $(dir $(shell find . -name '*.go' -path './tempodb/wal*/*' -type f | sort))))

# all other tests (excluding pkg & tempodb)
.PHONY: test-with-cover-others
test-with-cover-others: tools ## Run other tests with code coverage
	mkdir -p $(COVERAGE_DIR)
	$(GOTEST) $(GOTEST_OPT) -coverprofile=$(COVERAGE_DIR)/others.out $(shell go list $(sort $(dir $(OTHERS_SRC))))

##@ End to End Tests

# runs e2e tests in the top level integration/e2e directory
.PHONY: test-e2e
test-e2e: tools docker-tempo docker-tempo-query test-e2e-operations test-e2e-api test-e2e-limits test-e2e-metrics-generator test-e2e-storage test-e2e-util ## Run all e2e tests
	@echo "All e2e tests completed"

# runs only operations e2e tests
.PHONY: test-e2e-operations
test-e2e-operations: tools docker-tempo docker-tempo-query ## Run operations e2e tests
	$(GOTEST) -v $(GOTEST_OPT) ./integration/operations

# runs only api e2e tests
.PHONY: test-e2e-api
test-e2e-api: tools docker-tempo docker-tempo-query ## Run api e2e tests
	$(GOTEST) -v $(GOTEST_OPT) ./integration/api

## runs only poller integration tests
.PHONY: test-e2e-limits
test-e2e-limits: tools tools docker-tempo ## Run limits e2e tests
	$(GOTEST) -v $(GOTEST_OPT) ./integration/limits

# runs only metrics-generator integration tests
.PHONY: test-e2e-metrics-generator
test-e2e-metrics-generator: tools docker-tempo ## Run metrics-generator e2e tests
	$(GOTEST) -v $(GOTEST_OPT) ./integration/metrics-generator

# runs only ingest integration tests
.PHONY: test-e2e-storage
test-e2e-storage: tools docker-tempo ## Run storage e2e tests
	$(GOTEST) -v $(GOTEST_OPT) ./integration/storage

# runs only ingest integration tests
.PHONY: test-e2e-util
test-e2e-util: tools docker-tempo ## Run unit tests on the e2e test harness
	$(GOTEST) -v $(GOTEST_OPT) ./integration/util

# test-all use a docker image so build it first to make sure we're up to date
.PHONY: test-all
test-all: test-with-cover test-e2e ## Run all tests

# e2e test dirs are created by the host test process but their contents are written by
# Tempo containers running as UID 10001. Clean up in two steps: first empty the contents
# as UID 10001, then remove the now-empty dirs as the host user.
.PHONY: test-e2e-clean
test-e2e-clean: ## Remove leftover e2e test directories owned by Docker container UIDs
	docker run --rm -u 10001 \
		-v "$(shell pwd)/integration:/integration:z" \
		alpine find /integration -maxdepth 3 -name 'e2e_integration_test*' -type d \
		  -exec sh -c 'find "$$1" -mindepth 1 -delete' _ {} \;
	find $(shell pwd)/integration -maxdepth 3 -name 'e2e_integration_test*' -type d -prune -exec rm -rf '{}' +

##@ Linters/Formatters

.PHONY: fmt check-fmt
fmt: tools-image ## Format codebase with gofumpt and goimports
	@$(TOOLS_CMD) gofumpt -w $(FILES_TO_FMT)
	@$(TOOLS_CMD) goimports -w $(FILES_TO_FMT)

check-fmt: fmt
	@git diff --exit-code -- $(FILES_TO_FMT)

.PHONY: jsonnetfmt check-jsonnetfmt ## Check jsonnetfmt
jsonnetfmt: tools-image ## Format jsonnet codebase with jsonnetfmt
	@$(TOOLS_CMD) jsonnetfmt -i $(FILES_TO_JSONNETFMT)

check-jsonnetfmt: jsonnetfmt
	@git diff --exit-code -- $(FILES_TO_JSONNETFMT)

.PHONY: lint
lint: ## Lint codebase with golangci-lint
ifneq ($(base),)
	$(LINT_CMD) $(LINT) run --config .golangci.yml --new-from-rev=$(base)
else
	$(LINT_CMD) $(LINT) run --config .golangci.yml
endif

##@ Code Coverage

.PHONY: coverage-clean
coverage-clean: ## Clean coverage files
	rm -rf $(COVERAGE_DIR)

##@ Docker Images

.PHONY: docker-component 
docker-component: check-component exe # not intended to be used directly
	docker build -t grafana/$(COMPONENT) --build-arg=TARGETARCH=$(GOARCH) -f ./cmd/$(COMPONENT)/Dockerfile .
	docker tag grafana/$(COMPONENT) $(COMPONENT)

.PHONY: docker-component-multi
docker-component-multi: check-component # not intended to be used directly
	GOOS=linux GOARCH=amd64 make $(COMPONENT)
	GOOS=linux GOARCH=arm64 make $(COMPONENT)
	docker buildx build -t grafana/$(COMPONENT) --platform linux/amd64,linux/arm64 --output type=docker -f ./cmd/$(COMPONENT)/Dockerfile .

.PHONY: docker-component-debug
docker-component-debug: check-component exe-debug 
	docker build -t grafana/$(COMPONENT)-debug --build-arg=TARGETARCH=$(GOARCH) -f ./cmd/$(COMPONENT)/Dockerfile_debug .
	docker tag grafana/$(COMPONENT)-debug $(COMPONENT)-debug

.PHONY: docker-tempo 
docker-tempo: ## Build tempo docker image
	COMPONENT=tempo make docker-component

.PHONY: docker-tempo-multi
docker-tempo-multi: ## Build multiarch image locally, requires containerd image store
	COMPONENT=tempo make docker-component-multi

docker-tempo-debug: ## Build tempo debug docker image
	COMPONENT=tempo make docker-component-debug

.PHONY: docker-cli
docker-tempo-cli: ## Build tempo cli docker image
	COMPONENT=tempo-cli make docker-component

.PHONY: docker-tempo-query
docker-tempo-query: ## Build tempo query docker image
	COMPONENT=tempo-query make docker-component

.PHONY: docker-tempo-vulture
docker-tempo-vulture: ## Build tempo vulture docker image
	COMPONENT=tempo-vulture make docker-component

.PHONY: docker-tempo-vulture-multi
docker-tempo-vulture-multi: ## Build tempo vulture docker image
	COMPONENT=tempo-vulture make docker-component-multi

.PHONY: docker-images ## Build all docker images
docker-images: docker-tempo docker-tempo-query docker-tempo-vulture

.PHONY: check-component
check-component:
ifndef COMPONENT
	$(error COMPONENT variable was not defined)
endif

##@ Code Generation

PROTO_INTERMEDIATE_DIR = pkg/.patched-proto
# wiresmith protobuf compiler (https://github.com/grafana/wiresmith). Install
# the version matching go.mod: `go install github.com/grafana/wiresmith/cmd/wiresmith@$(go list -m -f '{{.Version}}' github.com/grafana/wiresmith)`.
WIRESMITH = wiresmith

.PHONY: gen-proto
gen-proto:  ## Generate proto files
	@echo --
	@echo -- Deleting existing
	@echo --
	rm -rf opentelemetry-proto
	rm -rf $(PROTO_INTERMEDIATE_DIR)
	find pkg/tempopb -name *.pb.go | xargs -L 1 -I rm
	# Here we avoid removing our tempo.proto and our frontend.proto due to reliance on the gogoproto bits.
	# .wiresmith-proto holds the wiresmith-annotated OTel proto sources (checked in) and must survive.
	find pkg/tempopb -name *.proto | grep -v tempo.proto | grep -v frontend.proto | grep -v backendwork.proto | grep -v .wiresmith-proto | xargs -L 1 -I rm

	@echo --
	@echo -- Copying to $(PROTO_INTERMEDIATE_DIR)
	@echo --
	git submodule update --init
	mkdir -p $(PROTO_INTERMEDIATE_DIR)
	cp -R opentelemetry-proto/opentelemetry/proto/* $(PROTO_INTERMEDIATE_DIR)

	@echo --
	@echo -- Editing proto
	@echo --

	@# Update package and types from opentelemetry.proto.* -> tempopb.*
	@# giving final types like "tempopb.common.v1.InstrumentationLibrary" which
	@# will not conflict with other usages of opentelemetry proto in downstream apps.
	find $(PROTO_INTERMEDIATE_DIR) -name "*.proto" | xargs -L 1 sed -i $(SED_OPTS) 's+ opentelemetry.proto+ tempopb+g'

	@# Update go_package
	find $(PROTO_INTERMEDIATE_DIR) -name "*.proto" | xargs -L 1 sed -i $(SED_OPTS) 's+go.opentelemetry.io/proto/otlp+github.com/grafana/tempo/pkg/tempopb+g'

	@# Update import paths
	find $(PROTO_INTERMEDIATE_DIR) -name "*.proto" | xargs -L 1 sed -i $(SED_OPTS) 's+import "opentelemetry/proto/+import "+g'

	@echo --
	@echo -- Gen proto --
	@echo --
	@# Generate patched OpenTelemetry protos (output to pkg/tempopb/) with wiresmith.
	@# Source of truth is pkg/tempopb/.wiresmith-proto: the .patched-proto output
	@# above plus wiresmith field annotations (pointer=true on repeated message
	@# fields to keep the gogo []*T shapes). When the OTel submodule changes,
	@# diff $(PROTO_INTERMEDIATE_DIR) against .wiresmith-proto and port the changes.
	$(WIRESMITH) --proto_path=pkg/tempopb/.wiresmith-proto --out=pkg/tempopb --module=github.com/grafana/tempo

	@# Generate Tempo protos with wiresmith. tempo.proto/backendwork.proto import
	@# the patched OTel protos, so a combined tree is assembled first. The two
	@# files are staged under a tempopb/ subdir so their path-relative import key
	@# is tempopb/tempo.proto (protoc/buf keying); with --out=pkg the
	@# source-relative output then lands in pkg/tempopb/.
	rm -rf pkg/.wiresmith-build
	mkdir -p pkg/.wiresmith-build/tempopb
	cp -R pkg/tempopb/.wiresmith-proto/* pkg/.wiresmith-build/
	cp pkg/tempopb/tempo.proto pkg/tempopb/backendwork.proto pkg/.wiresmith-build/tempopb/
	$(WIRESMITH) --proto_path=pkg/.wiresmith-build --out=pkg --module=github.com/grafana/tempo pkg/.wiresmith-build/tempopb/tempo.proto pkg/.wiresmith-build/tempopb/backendwork.proto
	rm -rf pkg/.wiresmith-build

	@# Generate backend proto with wiresmith. The file is staged as
	@# backend/v1.proto so the source-relative output lands at
	@# tempodb/backend/v1.pb.go (package backend), matching the old layout.
	rm -rf pkg/.wiresmith-build
	mkdir -p pkg/.wiresmith-build/backend
	cp tempodb/backend/v1/v1.proto pkg/.wiresmith-build/backend/v1.proto
	$(WIRESMITH) --proto_path=pkg/.wiresmith-build --out=tempodb --module=github.com/grafana/tempo -M "backend/v1.proto=github.com/grafana/tempo/tempodb/backend" pkg/.wiresmith-build/backend/v1.proto
	rm -rf pkg/.wiresmith-build

	@# Generate frontend proto with wiresmith. The dskit httpgrpc import is
	@# copied from vendor with its gogoproto lines stripped (wiresmith does not
	@# parse gogo options); the httpgrpc-typed fields ride through codegen via
	@# (wiresmith.options.customtype) envelopes, see
	@# modules/frontend/v1/frontendv1pb/httpgrpc_envelope.go.
	rm -rf pkg/.wiresmith-build
	mkdir -p pkg/.wiresmith-build/frontendv1pb pkg/.wiresmith-build/github.com/grafana/dskit/httpgrpc
	cp modules/frontend/v1/frontendv1pb/frontend.proto pkg/.wiresmith-build/frontendv1pb/
	sed '/gogoproto/d' vendor/github.com/grafana/dskit/httpgrpc/httpgrpc.proto > pkg/.wiresmith-build/github.com/grafana/dskit/httpgrpc/httpgrpc.proto
	$(WIRESMITH) --proto_path=pkg/.wiresmith-build --out=modules/frontend/v1 --module=github.com/grafana/tempo -M "frontendv1pb/frontend.proto=github.com/grafana/tempo/modules/frontend/v1/frontendv1pb" -M "github.com/grafana/dskit/httpgrpc/httpgrpc.proto=github.com/grafana/dskit/httpgrpc" pkg/.wiresmith-build/frontendv1pb/frontend.proto
	rm -rf pkg/.wiresmith-build

	rm -rf $(PROTO_INTERMEDIATE_DIR)

.PHONY: check-otel-proto-pin
check-otel-proto-pin: ## Verify the opentelemetry-proto submodule pin matches pkg/tempopb/.wiresmith-proto/OTEL_PROTO_PIN (no submodule checkout, no network)
	./tools/check-otel-proto-pin.sh

.PHONY: check-otel-proto-sync
check-otel-proto-sync: ## Verify pkg/tempopb/.wiresmith-proto/*.proto match the opentelemetry-proto submodule content, modulo wiresmith annotations (requires submodule checkout)
	./tools/check-otel-proto-sync.sh

.PHONY: gen-traceql 
gen-traceql: tools-image ## Generate traceql
	$(TOOLS_CMD) make gen-traceql-local

.PHONY: gen-traceql-local
gen-traceql-local: ## Generate traceq local
	goyacc -l -o pkg/traceql/expr.y.go pkg/traceql/expr.y && rm -f y.output

.PHONY: gen-parquet-query
gen-parquet-query:  ## Generate Parquet query 
	go run ./pkg/parquetquerygen/predicates.go > ./pkg/parquetquery/predicates.gen.go && go fmt ./pkg/parquetquery/predicates.gen.go

##@ Tempo tools
### Check vendored and generated files are up to date
.PHONY: vendor-check
vendor-check: check-otel-proto-pin check-otel-proto-sync gen-proto update-mod gen-traceql gen-parquet-query ## Keep up to date vendorized files
	# pkg/tempopb/, tempodb/backend/, and modules/frontend/v1/frontendv1pb/ are
	# the full set of gen-proto's Go output directories (see its four wiresmith
	# invocations in the "Gen proto" section above) - keep this pathspec in sync
	# with that target so a stale generated file always fails this check.
	git diff --exit-code -- **/go.sum **/go.mod vendor/ pkg/tempopb/ pkg/traceql/ tempodb/backend/ modules/frontend/v1/frontendv1pb/


### Tidy dependencies for tempo modules
.PHONY: update-mod 
update-mod: tools-update-mod ## Update module
	go mod vendor
	go mod tidy -e


### Release (intended to be used in the .github/workflows/release.yml)
$(GORELEASER):
	go install github.com/goreleaser/goreleaser/v2@v2.16.0

.PHONY: release
release: $(GORELEASER)  ## Release 
	$(GORELEASER) release --clean

.PHONY: release-snapshot
release-snapshot: $(GORELEASER) ## Release snapshot
	$(GORELEASER) release --skip=validate --clean --snapshot

##@ Docs
.PHONY: docs
docs: ## Generate docs
	docker pull ${DOCS_IMAGE}
	docker run -v ${PWD}/docs/sources/tempo:/hugo/content/docs/tempo/latest:z -p 3002:3002 --rm $(DOCS_IMAGE) /bin/bash -c 'mkdir -p content/docs/grafana/latest/ && touch content/docs/grafana/latest/menu.yaml && make server'

.PHONY: docs-test ## Generate docs tests
docs-test:
	docker pull ${DOCS_IMAGE}
	docker run -v ${PWD}/docs/sources/tempo:/hugo/content/docs/tempo/latest:z -p 3002:3002 --rm $(DOCS_IMAGE) /bin/bash -c 'mkdir -p content/docs/grafana/latest/ && touch content/docs/grafana/latest/menu.yaml && make prod'

.PHONY: generate-manifest
generate-manifest:  ## Generate manifest.md file
	GO111MODULE=on CGO_ENABLED=0 go run -v pkg/docsgen/generate_manifest.go

##@ Release

TEMPO_IMAGE_TAG ?=
.PHONY: bump-tempo-image-tag
bump-tempo-image-tag: ## Bump grafana/tempo image tag in docker-compose and jsonnet files. Usage: make bump-tempo-image-tag TEMPO_IMAGE_TAG=2.10.2
	@test -n "$(TEMPO_IMAGE_TAG)" || (echo "ERROR: TEMPO_IMAGE_TAG is required. Usage: make bump-tempo-image-tag TEMPO_IMAGE_TAG=<tag>"; exit 1)
	@echo "$(TEMPO_IMAGE_TAG)" | grep -qE '^(latest|[0-9]+\.[0-9]+\.[0-9]+)$$' || (echo "ERROR: TEMPO_IMAGE_TAG must be 'latest' or a version like '2.10.2', got: $(TEMPO_IMAGE_TAG)"; exit 1)
	@echo "Replacing grafana/tempo:<any> -> grafana/tempo:$(TEMPO_IMAGE_TAG)"
	@find . \
		-not -path './.claude/*' \
		-not -path './.git/*' \
		-not -path './vendor/*' \
		-not -path './.worktrees/*' \
		\( -name "*.yaml" -o -name "*.libsonnet" -o -name "*.jsonnet" -o -name "*.md" \) \
		-print0 \
		| xargs -0 grep -l "grafana/tempo:" \
		| tr '\n' '\0' \
		| xargs -0 -I{} sed -i $(SED_OPTS) -E "s#grafana/tempo:(latest|[0-9]+\.[0-9]+\.[0-9]+)#grafana/tempo:$(TEMPO_IMAGE_TAG)#g" {}
	@echo "Done. Re-run 'make jsonnet' to regenerate compiled jsonnet if needed."

##@ jsonnet
.PHONY: jsonnet jsonnet-check jsonnet-test
jsonnet: tools-image ## Generate jsonnet
	$(TOOLS_CMD) make -C operations/jsonnet-compiled gen

jsonnet-check: tools-image ## Check jsonnet
	$(TOOLS_CMD) make -C operations/jsonnet-compiled check

jsonnet-test: tools-image ## Test jsonnet
	$(TOOLS_CMD) make -C operations/jsonnet/microservices test

### tempo-mixin
.PHONY: jsonnetfmt tempo-mixin tempo-mixin-check
tempo-mixin: tools-image
	$(TOOLS_CMD) make -C operations/tempo-mixin all

tempo-mixin-check: tools-image
	$(TOOLS_CMD) make -C operations/tempo-mixin check

##@ Changelog

# chloggen is built from the tools module and run from the repo root so it can
# find .chloggen/ and CHANGELOG.md. A .chloggen/config.yaml is used only if it
# exists. chlog-new defaults the entry filename to the current branch; override
# with FILENAME=... . Pass VERSION=vX.Y.Z to chlog-update.
CHLOGGEN ?= $(CURDIR)/bin/chloggen
CHLOGGEN_CONFIG := .chloggen/config.yaml
CHLOGGEN_CONFIG_ARG := $(if $(wildcard $(CHLOGGEN_CONFIG)),--config $(CHLOGGEN_CONFIG),)
CHLOG_BRANCH := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null)
CHLOG_FILENAME := $(if $(FILENAME),$(FILENAME),$(CHLOG_BRANCH))
CHLOG_EDITOR ?= $${VISUAL:-$${EDITOR:-vi}}
CHLOG_EDIT ?= 1

.PHONY: $(CHLOGGEN)
$(CHLOGGEN):
	cd $(CURDIR)/tools/chloggen && go build -o $(CHLOGGEN) .

.PHONY: chlog-new
chlog-new: $(CHLOGGEN) ## Create a new changelog entry under .chloggen/ (defaults to branch name; override with FILENAME=...)
	@if [ -z "$(CHLOG_FILENAME)" ] || [ "$(CHLOG_FILENAME)" = "HEAD" ] || \
	    [ "$(CHLOG_FILENAME)" = "main" ] || [ "$(CHLOG_FILENAME)" = "master" ]; then \
	  echo "Cannot default the changelog filename from branch '$(CHLOG_BRANCH)'; pass FILENAME=<name>."; \
	  exit 1; \
	fi
	@output="$$( $(CHLOGGEN) new $(CHLOGGEN_CONFIG_ARG) --filename "$(CHLOG_FILENAME)" )"; \
	status=$$?; \
	[ -z "$$output" ] || printf '%s\n' "$$output"; \
	if [ $$status -ne 0 ]; then exit $$status; fi; \
	entry="$$(printf '%s\n' "$$output" | awk '/^Changelog entry template copied to: /{sub(/^Changelog entry template copied to: /, ""); print; exit}')"; \
	if [ -z "$$entry" ]; then echo "Could not determine changelog entry path." >&2; exit 1; fi; \
	if [ "$(CHLOG_EDIT)" != "0" ] && [ -t 0 ] && [ -t 1 ]; then \
	  $(CHLOG_EDITOR) "$$entry"; \
	else \
	  echo "Edit $$entry manually."; \
	fi

.PHONY: chlog-validate
chlog-validate: $(CHLOGGEN) ## Validate the pending changelog entries
	$(CHLOGGEN) validate $(CHLOGGEN_CONFIG_ARG)

.PHONY: chlog-preview
chlog-preview: $(CHLOGGEN) ## Render the pending changelog entries to stdout
	$(CHLOGGEN) update $(CHLOGGEN_CONFIG_ARG) --dry

.PHONY: chlog-update
chlog-update: $(CHLOGGEN) ## Collate pending entries into CHANGELOG.md (VERSION=vX.Y.Z)
	$(CHLOGGEN) update $(CHLOGGEN_CONFIG_ARG) --version "$(VERSION)"

# Import fragments
include build/tools.mk
