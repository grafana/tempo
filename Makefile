# Adapted from https://www.thapaliya.com/en/writings/well-documented-makefiles/
.PHONY: help
help:  ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.DEFAULT_GOAL:=help

# Version number
VERSION=$(shell ./tools/version-tag.sh | cut -d, -f 1)

GIT_REVISION := $(shell git rev-parse --short HEAD)
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD)

GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

GOPATH := $(shell go env GOPATH)
GORELEASER := $(GOPATH)/bin/goreleaser

# Build Images
DOCKER_PROTOBUF_IMAGE ?= otel/build-protobuf:0.23.0
LOKI_BUILD_IMAGE ?= grafana/loki-build-image:0.33.2
DOCS_IMAGE ?= grafana/docs-base:latest

# More exclusions can be added similar with: -not -path './testbed/*'
ALL_SRC := $(shell find . -name '*.go' \
								-not -path './tools*/*' \
								-not -path './vendor*/*' \
								-not -path './integration/*' \
								-not -path './cmd/tempo-serverless/*' \
                                -type f | sort)

# ALL_SRC but without pkg and tempodb packages
OTHERS_SRC := $(shell find . -name '*.go' \
								-not -path './tools*/*' \
								-not -path './vendor*/*' \
								-not -path './integration/*' \
								-not -path './cmd/tempo-serverless/*' \
								-not -path './pkg*/*' \
								-not -path './tempodb*/*' \
                                -type f | sort)

# All source code and documents. Used in spell check.
ALL_DOC := $(shell find . \( -name "*.md" -o -name "*.yaml" \) \
                                -type f | sort)

# ALL_PKGS is used with 'go cover'
ALL_PKGS := $(shell go list $(sort $(dir $(ALL_SRC))))

GO_OPT= -mod vendor -ldflags "-X main.Branch=$(GIT_BRANCH) -X main.Revision=$(GIT_REVISION) -X main.Version=$(VERSION)"
ifeq ($(BUILD_DEBUG), 1)
	GO_OPT+= -gcflags="all=-N -l"
endif

GOTEST_OPT?= -race -timeout 25m -count=1 -v
GOTEST_OPT_WITH_COVERAGE = $(GOTEST_OPT) -cover
GOTEST=gotestsum --format=testname --
LINT=golangci-lint

UNAME := $(shell uname -s)
ifeq ($(UNAME), Darwin)
    SED_OPTS := ''
endif

FILES_TO_FMT=$(shell find . -type d \( -path ./vendor -o -path ./opentelemetry-proto -o -path ./vendor-fix \) -prune -o -name '*.go' -not -name "*.pb.go" -not -name '*.y.go' -not -name '*.gen.go' -print)
FILES_TO_JSONNETFMT=$(shell find ./operations/jsonnet ./operations/tempo-mixin -type f \( -name '*.libsonnet' -o -name '*.jsonnet' \) -not -path "*/vendor/*" -print)

##@ Building
.PHONY: tempo 	
tempo: ## Build tempo
	GO111MODULE=on CGO_ENABLED=0 go build $(GO_OPT) -o ./bin/$(GOOS)/tempo-$(GOARCH) $(BUILD_INFO) ./cmd/tempo 

.PHONY: tempo-query
tempo-query: ## Build tempo-query
	GO111MODULE=on CGO_ENABLED=0 go build $(GO_OPT) -o ./bin/$(GOOS)/tempo-query-$(GOARCH) $(BUILD_INFO) ./cmd/tempo-query

.PHONY: tempo-cli
tempo-cli: ## Build tempo-cli
	GO111MODULE=on CGO_ENABLED=0 go build $(GO_OPT) -o ./bin/$(GOOS)/tempo-cli-$(GOARCH) $(BUILD_INFO) ./cmd/tempo-cli

.PHONY: tempo-vulture  ## Build tempo-vulture
tempo-vulture:
	GO111MODULE=on CGO_ENABLED=0 go build $(GO_OPT) -o ./bin/$(GOOS)/tempo-vulture-$(GOARCH) $(BUILD_INFO) ./cmd/tempo-vulture

.PHONY: exe  ## Build exe
exe:
	GOOS=linux $(MAKE) $(COMPONENT)

.PHONY: exe-debug  ## Build exe-debug
exe-debug:
	BUILD_DEBUG=1 GOOS=linux $(MAKE) $(COMPONENT)

##@  Testin' and Lintin'

.PHONY: test
test: ## Run tests
	$(GOTEST) $(GOTEST_OPT) $(ALL_PKGS)

.PHONY: benchmark
benchmark: tools ## Run benchmarks
	$(GOTEST) -bench=. -run=notests $(ALL_PKGS)

# Not used in CI, tests are split in pkg, tempodb, tempodb-wal and others in CI jobs
.PHONY: test-with-cover 
test-with-cover: tools test-serverless ## Run tests with code coverage
	$(GOTEST) $(GOTEST_OPT_WITH_COVERAGE) $(ALL_PKGS)

# tests in pkg
.PHONY: test-with-cover-pkg 
test-with-cover-pkg: tools  ##  Run Tempo packages' tests with code coverage
	$(GOTEST) $(GOTEST_OPT_WITH_COVERAGE) $(shell go list $(sort $(dir $(shell find . -name '*.go' -path './pkg*/*' -type f | sort))))

# tests in tempodb (excluding tempodb/wal)
.PHONY: test-with-cover-tempodb
test-with-cover-tempodb: tools ## Run tempodb tests with code coverage
	GOMEMLIMIT=6GiB $(GOTEST) $(GOTEST_OPT_WITH_COVERAGE) $(shell go list $(sort $(dir $(shell find . -name '*.go'  -not -path './tempodb/wal*/*' -path './tempodb*/*' -type f | sort))))

# tests in tempodb/wal
.PHONY: test-with-cover-tempodb-wal
test-with-cover-tempodb-wal: tools  ## Test tempodb/wal with code coverage
	$(GOTEST) $(GOTEST_OPT_WITH_COVERAGE) $(shell go list $(sort $(dir $(shell find . -name '*.go' -path './tempodb/wal*/*' -type f | sort))))

# all other tests (excluding pkg & tempodb)
.PHONY: test-with-cover-others
test-with-cover-others: tools test-serverless ## Run other tests with code coverage
	$(GOTEST) $(GOTEST_OPT_WITH_COVERAGE) $(shell go list $(sort $(dir $(OTHERS_SRC))))

# runs e2e tests in the top level integration/e2e directory
.PHONY: test-e2e
test-e2e: tools docker-tempo docker-tempo-query  ## Run end to end tests
	$(GOTEST) -v $(GOTEST_OPT) ./integration/e2e

# runs only serverless e2e tests
.PHONY: test-e2e-serverless
test-e2e-serverless: tools docker-tempo docker-serverless ## Run serverless end to end tests
	$(GOTEST) -v $(GOTEST_OPT) ./integration/e2e/serverless

# runs only deployment modes e2e tests
.PHONY: test-e2e-deployments
test-e2e-deployments: tools docker-tempo docker-tempo-query ## Run end to end tests for deployments
	$(GOTEST) -v $(GOTEST_OPT) ./integration/e2e/deployments

# runs only poller integration tests
.PHONY: test-integration-poller
test-integration-poller: tools ## Run poller integration tests
	$(GOTEST) -v $(GOTEST_OPT) ./integration/poller

# test-all/bench use a docker image so build it first to make sure we're up to date
.PHONY: test-all ## Run all tests
test-all: test-with-cover test-e2e test-e2e-serverless test-e2e-deployments test-integration-poller

.PHONY: test-bench
test-bench: tools docker-tempo ## Run all benchmarks
	$(GOTEST) -v $(GOTEST_OPT) ./integration/bench

.PHONY: fmt check-fmt
fmt: tools-image ## Check fmt
	@$(TOOLS_CMD) gofumpt -w $(FILES_TO_FMT)
	@$(TOOLS_CMD) goimports -w $(FILES_TO_FMT)

check-fmt: fmt
	@git diff --exit-code -- $(FILES_TO_FMT)

.PHONY: jsonnetfmt check-jsonnetfmt ## Check jsonnetfmt
jsonnetfmt: tools-image
	@$(TOOLS_CMD) jsonnetfmt -i $(FILES_TO_JSONNETFMT)

check-jsonnetfmt: jsonnetfmt
	@git diff --exit-code -- $(FILES_TO_JSONNETFMT)

.PHONY: lint
lint: # linting
ifneq ($(base),)
	$(LINT_CMD) $(LINT) run --config .golangci.yml --new-from-rev=$(base)
else
	$(LINT_CMD) $(LINT) run --config .golangci.yml
endif

##@ Docker Images

.PHONY: docker-component 
docker-component: check-component exe # not intended to be used directly
	docker build -t grafana/$(COMPONENT) --build-arg=TARGETARCH=$(GOARCH) -f ./cmd/$(COMPONENT)/Dockerfile .
	docker tag grafana/$(COMPONENT) $(COMPONENT)

.PHONY: docker-component-multi
docker-component-multi: check-component # not intended to be used directly
	GOOS=linux GOARCH=amd64 $(MAKE) $(COMPONENT)
	GOOS=linux GOARCH=arm64 $(MAKE) $(COMPONENT)
	docker buildx build -t grafana/$(COMPONENT) --platform linux/amd64,linux/arm64 --output type=docker -f ./cmd/$(COMPONENT)/Dockerfile .

.PHONY: docker-component-debug
docker-component-debug: check-component exe-debug 
	docker build -t grafana/$(COMPONENT)-debug --build-arg=TARGETARCH=$(GOARCH) -f ./cmd/$(COMPONENT)/Dockerfile_debug .
	docker tag grafana/$(COMPONENT)-debug $(COMPONENT)-debug

.PHONY: docker-tempo 
docker-tempo: ## Build tempo docker image
	COMPONENT=tempo $(MAKE) docker-component

.PHONY: docker-tempo-multi
docker-tempo-multi: ## Build multiarch image locally, requires containerd image store
	COMPONENT=tempo $(MAKE) docker-component-multi

docker-tempo-debug: ## Build tempo debug docker image
	COMPONENT=tempo $(MAKE) docker-component-debug

.PHONY: docker-cli
docker-tempo-cli: ## Build tempo cli docker image
	COMPONENT=tempo-cli $(MAKE) docker-component

.PHONY: docker-tempo-query
docker-tempo-query: ## Build tempo query docker image
	COMPONENT=tempo-query $(MAKE) docker-component

.PHONY: docker-tempo-vulture
docker-tempo-vulture: ## Build tempo vulture docker image
	COMPONENT=tempo-vulture $(MAKE) docker-component

.PHONY: docker-images ## Build all docker images
docker-images: docker-tempo docker-tempo-query docker-tempo-vulture

.PHONY: check-component
check-component:
ifndef COMPONENT
	$(error COMPONENT variable was not defined)
endif

##@ Gen Proto

PROTOC = docker run --rm -u ${shell id -u} -v${PWD}:${PWD} -w${PWD} ${DOCKER_PROTOBUF_IMAGE} --proto_path=${PWD}
PROTO_INTERMEDIATE_DIR = pkg/.patched-proto
PROTO_INCLUDES = -I$(PROTO_INTERMEDIATE_DIR)
PROTO_GEN = $(PROTOC) $(PROTO_INCLUDES) --gogofaster_out=plugins=grpc,paths=source_relative:$(2) $(1)
PROTO_GEN_WITH_VENDOR = $(PROTOC) $(PROTO_INCLUDES) -Ivendor -Ivendor/github.com/gogo/protobuf --gogofaster_out=plugins=grpc,paths=source_relative:$(2) $(1)
PROTO_GEN_WITHOUT_RELATIVE = $(PROTOC) $(PROTO_INCLUDES) --gogofaster_out=plugins=grpc:$(2) $(1)

.PHONY: gen-proto
gen-proto:  ## Generate proto files
	@echo --
	@echo -- Deleting existing
	@echo --
	rm -rf opentelemetry-proto
	rm -rf $(PROTO_INTERMEDIATE_DIR)
	find pkg/tempopb -name *.pb.go | xargs -L 1 -I rm
	# Here we avoid removing our tempo.proto and our frontend.proto due to reliance on the gogoproto bits.
	find pkg/tempopb -name *.proto | grep -v tempo.proto | grep -v frontend.proto | xargs -L 1 -I rm

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
	$(call PROTO_GEN,$(PROTO_INTERMEDIATE_DIR)/common/v1/common.proto,./pkg/tempopb/)
	$(call PROTO_GEN,$(PROTO_INTERMEDIATE_DIR)/resource/v1/resource.proto,./pkg/tempopb/)
	$(call PROTO_GEN,$(PROTO_INTERMEDIATE_DIR)/trace/v1/trace.proto,./pkg/tempopb/)
	$(call PROTO_GEN,pkg/tempopb/tempo.proto,./)
	$(call PROTO_GEN_WITHOUT_RELATIVE,tempodb/backend/v1/v1.proto,./)
	$(call PROTO_GEN_WITH_VENDOR,modules/frontend/v1/frontendv1pb/frontend.proto,./)

	rm -rf $(PROTO_INTERMEDIATE_DIR)

##@ Gen Traceql

.PHONY: gen-traceql 
gen-traceql: ## Generate traceql 
	docker run --rm -v${PWD}:/src/loki ${LOKI_BUILD_IMAGE} gen-traceql-local

.PHONY: gen-traceql-local 
gen-traceql-local: ## Generate traceq local
	goyacc -o pkg/traceql/expr.y.go pkg/traceql/expr.y && rm y.output


##@ Gen Parquet-Query

.PHONY: gen-parquet-query
gen-parquet-query:  ## Generate Parquet query 
	go run ./pkg/parquetquerygen/predicates.go > ./pkg/parquetquery/predicates.gen.go

##@ Tempo tools
### Check vendored and generated files are up to date
.PHONY: vendor-check
vendor-check: gen-proto update-mod gen-traceql gen-parquet-query ## Keep up to date vendorized files
	git diff --exit-code -- **/go.sum **/go.mod vendor/ pkg/tempopb/ pkg/traceql/


### Tidy dependencies for tempo and tempo-serverless modules
.PHONY: update-mod 
update-mod: tools-update-mod ## Update module
	go mod vendor
	go mod tidy -e
	$(MAKE) -C cmd/tempo-serverless update-mod


### Release (intended to be used in the .github/workflows/release.yml)
$(GORELEASER):
	go install github.com/goreleaser/goreleaser@v1.25.1

.PHONY: release
release: $(GORELEASER)  ## Release 
	$(GORELEASER) release --rm-dist 

.PHONY: release-snapshot
release-snapshot: $(GORELEASER) ## Release snapshot
	$(GORELEASER) release --skip-validate --rm-dist --snapshot

##@ Docs
.PHONY: docs
docs: ## Generate docs
	docker pull ${DOCS_IMAGE}
	docker run -v ${PWD}/docs/sources/tempo:/hugo/content/docs/tempo/latest:z -p 3002:3002 --rm $(DOCS_IMAGE) /bin/bash -c 'mkdir -p content/docs/grafana/latest/ && touch content/docs/grafana/latest/menu.yaml && make server'

.PHONY: docs-test ## Generate docs tests
docs-test:
	docker pull ${DOCS_IMAGE}
	docker run -v ${PWD}/docs/sources/tempo:/hugo/content/docs/tempo/latest:z -p 3002:3002 --rm $(DOCS_IMAGE) /bin/bash -c 'mkdir -p content/docs/grafana/latest/ && touch content/docs/grafana/latest/menu.yaml && make prod'

##@ jsonnet
.PHONY: jsonnet jsonnet-check jsonnet-test
jsonnet: tools-image ## Generate jsonnet
	$(TOOLS_CMD) $(MAKE) -C operations/jsonnet-compiled/util gen

jsonnet-check: tools-image ## Check jsonnet
	$(TOOLS_CMD) $(MAKE) -C operations/jsonnet-compiled/util check

jsonnet-test: tools-image ## Test jsonnet
	$(TOOLS_CMD) $(MAKE) -C operations/jsonnet/microservices test

##@ serverless
.PHONY: docker-serverless test-serverless
docker-serverless: ## Build docker Tempo serverless
	$(MAKE) -C cmd/tempo-serverless build-docker

test-serverless: ## Run Tempo serverless tests
	$(MAKE) -C cmd/tempo-serverless test

### tempo-mixin
.PHONY: tempo-mixin tempo-mixin-check
tempo-mixin: tools-image
	$(TOOLS_CMD) $(MAKE) -C operations/tempo-mixin all

tempo-mixin-check: tools-image
	$(TOOLS_CMD) $(MAKE) -C operations/tempo-mixin check

.PHONY: generate-manifest
generate-manifest:
	GO111MODULE=on CGO_ENABLED=0 go run -v pkg/docsgen/generate_manifest.go

# Import fragments
include build/tools.mk
