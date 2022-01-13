# Version number
VERSION=$(shell ./tools/image-tag | cut -d, -f 1)

GIT_REVISION := $(shell git rev-parse --short HEAD)
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD)

GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

GOPATH := $(shell go env GOPATH)
GORELEASER := $(GOPATH)/bin/goreleaser

# More exclusions can be added similar with: -not -path './testbed/*'
ALL_SRC := $(shell find . -name '*.go' \
								-not -path './vendor*/*' \
								-not -path './integration/*' \
								-not -path './cmd/tempo-serverless/*' \
                                -type f | sort)

# All source code and documents. Used in spell check.
ALL_DOC := $(shell find . \( -name "*.md" -o -name "*.yaml" \) \
                                -type f | sort)

# ALL_PKGS is used with 'go cover'
ALL_PKGS := $(shell go list $(sort $(dir $(ALL_SRC))))

GO_OPT= -mod vendor -ldflags "-X main.Branch=$(GIT_BRANCH) -X main.Revision=$(GIT_REVISION) -X main.Version=$(VERSION)"
GOTEST_OPT?= -race -timeout 13m -count=1
GOTEST_OPT_WITH_COVERAGE = $(GOTEST_OPT) -cover
GOTEST=go test
LINT=golangci-lint

UNAME := $(shell uname -s)
ifeq ($(UNAME), Darwin)
    SED_OPTS := ''
endif

FILES_TO_FMT=$(shell find . -type d \( -path ./vendor -o -path ./opentelemetry-proto -o -path ./vendor-fix \) -prune -o -name '*.go' -not -name "*.pb.go" -print)

### Build

.PHONY: tempo
tempo:
	GO111MODULE=on CGO_ENABLED=0 go build $(GO_OPT) -o ./bin/$(GOOS)/tempo-$(GOARCH) $(BUILD_INFO) ./cmd/tempo

.PHONY: tempo-query
tempo-query:
	GO111MODULE=on CGO_ENABLED=0 go build $(GO_OPT) -o ./bin/$(GOOS)/tempo-query-$(GOARCH) $(BUILD_INFO) ./cmd/tempo-query

.PHONY: tempo-cli
tempo-cli:
	GO111MODULE=on CGO_ENABLED=0 go build $(GO_OPT) -o ./bin/$(GOOS)/tempo-cli-$(GOARCH) $(BUILD_INFO) ./cmd/tempo-cli

.PHONY: tempo-vulture
tempo-vulture:
	GO111MODULE=on CGO_ENABLED=0 go build $(GO_OPT) -o ./bin/$(GOOS)/tempo-vulture-$(GOARCH) $(BUILD_INFO) ./cmd/tempo-vulture

.PHONY: exe
exe:
	GOOS=linux $(MAKE) $(COMPONENT)

### Testin' and Lintin'

.PHONY: test
test:
	$(GOTEST) $(GOTEST_OPT) $(ALL_PKGS)

.PHONY: benchmark
benchmark:
	$(GOTEST) -bench=. -run=notests $(ALL_PKGS)

.PHONY: test-with-cover
test-with-cover: test-serverless
	$(GOTEST) $(GOTEST_OPT_WITH_COVERAGE) $(ALL_PKGS)

.PHONY: test-e2e
test-e2e: docker-tempo docker-serverless
	$(GOTEST) -v $(GOTEST_OPT) ./integration/e2e

# test-all/bench use a docker image so build it first to make sure we're up to date
.PHONY: test-all
test-all: test-with-cover test-e2e

.PHONY: test-bench
test-bench: docker-tempo
	$(GOTEST) -v $(GOTEST_OPT) ./integration/bench

.PHONY: fmt
fmt:
	@gofmt -s -w $(FILES_TO_FMT)
	@goimports -w $(FILES_TO_FMT)

.PHONY: lint
lint:
	$(LINT) run

### Docker Images

.PHONY: docker-component # Not intended to be used directly
docker-component: check-component exe
	docker build -t grafana/$(COMPONENT) --build-arg=TARGETARCH=$(GOARCH) -f ./cmd/$(COMPONENT)/Dockerfile .
	docker tag grafana/$(COMPONENT) $(COMPONENT)

.PHONY: docker-tempo
docker-tempo:
	COMPONENT=tempo $(MAKE) docker-component

.PHONY: docker-tempo-query
docker-tempo-query:
	COMPONENT=tempo-query $(MAKE) docker-component

.PHONY: docker-tempo-vulture
docker-tempo-vulture:
	COMPONENT=tempo-vulture $(MAKE) docker-component

.PHONY: docker-images
docker-images: docker-tempo docker-tempo-query docker-tempo-vulture

.PHONY: check-component
check-component:
ifndef COMPONENT
	$(error COMPONENT variable was not defined)
endif

### Dependencies
DOCKER_PROTOBUF ?= otel/build-protobuf:0.2.1
PROTOC = docker run --rm -u ${shell id -u} -v${PWD}:${PWD} -w${PWD} ${DOCKER_PROTOBUF} --proto_path=${PWD}
PROTO_INTERMEDIATE_DIR = pkg/.patched-proto
PROTO_INCLUDES = -I$(PROTO_INTERMEDIATE_DIR)
PROTO_GEN = $(PROTOC) $(PROTO_INCLUDES) --gogofaster_out=plugins=grpc,paths=source_relative:$(2) $(1)

.PHONY: gen-proto
gen-proto: 
	@echo --
	@echo -- Deleting existing
	@echo --
	rm -rf opentelemetry-proto
	rm -rf $(PROTO_INTERMEDIATE_DIR)
	find pkg/tempopb -name *.pb.go | xargs -L 1 -I rm
	find pkg/tempopb -name *.proto | grep -v tempo.proto | xargs -L 1 -I rm

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
	find $(PROTO_INTERMEDIATE_DIR) -name "*.proto" | xargs -L 1 sed -i $(SED_OPTS) 's+github.com/open-telemetry/opentelemetry-proto/gen/go+github.com/grafana/tempo/pkg/tempopb+g'

	@# Update import paths
	find $(PROTO_INTERMEDIATE_DIR) -name "*.proto" | xargs -L 1 sed -i $(SED_OPTS) 's+import "opentelemetry/proto/+import "+g'

	@echo --
	@echo -- Gen proto -- 
	@echo --
	$(call PROTO_GEN,$(PROTO_INTERMEDIATE_DIR)/common/v1/common.proto,./pkg/tempopb/)
	$(call PROTO_GEN,$(PROTO_INTERMEDIATE_DIR)/resource/v1/resource.proto,./pkg/tempopb/)
	$(call PROTO_GEN,$(PROTO_INTERMEDIATE_DIR)/trace/v1/trace.proto,./pkg/tempopb/)
	$(call PROTO_GEN,pkg/tempopb/tempo.proto,./)

	rm -rf $(PROTO_INTERMEDIATE_DIR)

.PHONY: gen-flat
gen-flat:
	# -o /pkg generates into same folder as tempo.fbs for simpler imports.
	docker run --rm -v${PWD}:/opt/src neomantra/flatbuffers flatc --go -o /opt/src/pkg /opt/src/pkg/tempofb/tempo.fbs

### Check vendored files and generated proto
.PHONY: vendor-check
vendor-check: gen-proto gen-flat
	go mod vendor
	go mod tidy -e
	git diff --exit-code -- go.sum go.mod vendor/ pkg/tempopb/ pkg/tempofb/


### Release (intended to be used in the .github/workflows/release.yml)
$(GORELEASER):
	go install github.com/goreleaser/goreleaser@latest

release: $(GORELEASER)
	$(GORELEASER) build --skip-validate --rm-dist
	$(GORELEASER) release --rm-dist

release-snapshot: $(GORELEASER)
	$(GORELEASER) build --skip-validate --rm-dist --snapshot

### Docs
DOCS_IMAGE = grafana/docs-base:latest

.PHONY: docs
docs:
	docker pull ${DOCS_IMAGE}
	docker run -v ${PWD}/docs/tempo/website:/hugo/content/docs/tempo/latest:z -p 3002:3002 --rm $(DOCS_IMAGE) /bin/bash -c 'mkdir -p content/docs/grafana/latest/ && touch content/docs/grafana/latest/menu.yaml && make server'

.PHONY: docs-test
docs-test:
	docker pull ${DOCS_IMAGE}
	docker run -v ${PWD}/docs/tempo/website:/hugo/content/docs/tempo/latest:z -p 3002:3002 --rm $(DOCS_IMAGE) /bin/bash -c 'mkdir -p content/docs/grafana/latest/ && touch content/docs/grafana/latest/menu.yaml && make prod'

### kube-manifests
.PHONY: kube-manifests kube-manifests-check
kube-manifests:
	$(MAKE) -C operations/kube-manifests/util gen

kube-manifests-check:
	$(MAKE) -C operations/kube-manifests/util check


### serverless
.PHONY: docker-serverless test-serverless
docker-serverless:
	$(MAKE) -C cmd/tempo-serverless build-docker

test-serverless:
	$(MAKE) -C cmd/tempo-serverless test

### tempo-mixin
.PHONY: tempo-mixin tempo-mixin-check
tempo-mixin:
	$(MAKE) -C operations/tempo-mixin all

tempo-mixin-check:
	$(MAKE) -C operations/tempo-mixin check

### drone
.PHONY: drone drone-jsonnet drone-signature
# this requires the drone-cli https://docs.drone.io/cli/install/
drone:
	# piggyback on Loki's build image, this image contains a newer version of drone-cli than is
	# released currently (1.4.0). The newer version of drone-clie keeps drone.yml human-readable.
	# This will run 'make drone-jsonnet' from within the container
	docker run --rm -v $(shell pwd):/src/loki grafana/loki-build-image:0.15.0 drone-jsonnet

	drone lint .drone/drone.yml
	@make drone-signature

drone-jsonnet:
	drone jsonnet --stream --format --source .drone/drone.jsonnet --target .drone/drone.yml

drone-signature:
ifndef DRONE_TOKEN
	$(error DRONE_TOKEN is not set, visit https://drone.grafana.net/account)
endif
	DRONE_SERVER=https://drone.grafana.net drone sign --save grafana/tempo .drone/drone.yml
