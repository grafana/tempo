GOPATH := $(shell go env GOPATH)
GORELEASER := $(GOPATH)/bin/goreleaser

# More exclusions can be added similar with: -not -path './testbed/*'
ALL_SRC := $(shell find . -name '*.go' \
								-not -path './vendor/*' \
								-not -path './integration/*' \
                                -type f | sort)

# All source code and documents. Used in spell check.
ALL_DOC := $(shell find . \( -name "*.md" -o -name "*.yaml" \) \
                                -type f | sort)

# ALL_PKGS is used with 'go cover'
ALL_PKGS := $(shell go list $(sort $(dir $(ALL_SRC))))

GO_OPT= -mod vendor
GOTEST_OPT?= -race -timeout 5m -count=1
GOTEST_OPT_WITH_COVERAGE = $(GOTEST_OPT) -cover
GOTEST=go test
LINT=golangci-lint

UNAME := $(shell uname -s)
ifeq ($(UNAME), Darwin)
    SED_OPTS := ''
endif

### Build

.PHONY: tempo
tempo:
	GO111MODULE=on CGO_ENABLED=0 go build $(GO_OPT) -o ./bin/$(GOOS)/tempo $(BUILD_INFO) ./cmd/tempo

.PHONY: tempo-query
tempo-query:
	GO111MODULE=on CGO_ENABLED=0 go build $(GO_OPT) -o ./bin/$(GOOS)/tempo-query $(BUILD_INFO) ./cmd/tempo-query

.PHONY: tempo-cli
tempo-cli:
	GO111MODULE=on CGO_ENABLED=0 go build $(GO_OPT) -o ./bin/$(GOOS)/tempo-cli $(BUILD_INFO) ./cmd/tempo-cli

.PHONY: tempo-vulture
tempo-vulture:
	GO111MODULE=on CGO_ENABLED=0 go build $(GO_OPT) -o ./bin/$(GOOS)/tempo-vulture $(BUILD_INFO) ./cmd/tempo-vulture

.PHONY: exe
exe:
	GOOS=linux $(MAKE) $(COMPONENT)
	cp ./bin/linux/$(COMPONENT) ./cmd/$(COMPONENT)/

### Testin' and Lintin'

.PHONY: test
test:
	$(GOTEST) $(GOTEST_OPT) $(ALL_PKGS)

.PHONY: benchmark
benchmark:
	$(GOTEST) -bench=. -run=notests $(ALL_PKGS)

.PHONY: test-with-cover
test-with-cover: 
	$(GOTEST) $(GOTEST_OPT_WITH_COVERAGE) $(ALL_PKGS)

# test-all includes integration tests so we build our docker image first
.PHONY: test-all
test-all: docker-tempo test-with-cover
	$(GOTEST) $(GOTEST_OPT_WITH_COVERAGE) ./integration/e2e

.PHONY: lint
lint:
	$(LINT) run

### Docker Images

.PHONY: docker-component # Not intended to be used directly
docker-component: check-component exe
	docker build -t grafana/$(COMPONENT) -f ./cmd/$(COMPONENT)/Dockerfile .
	docker tag grafana/$(COMPONENT) $(COMPONENT)
	rm ./cmd/$(COMPONENT)/$(COMPONENT)

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

.PHONY: gen-proto
gen-proto:
	git submodule init
	git submodule update
	rm -rf ./vendor/github.com/open-telemetry/opentelemetry-proto
	protoc -I opentelemetry-proto/ opentelemetry-proto/opentelemetry/proto/common/v1/common.proto --gogofaster_out=plugins=grpc:./vendor
	protoc -I opentelemetry-proto/ opentelemetry-proto/opentelemetry/proto/resource/v1/resource.proto --gogofaster_out=plugins=grpc:./vendor
	# protoc -I opentelemetry-proto/ opentelemetry-proto/opentelemetry/proto/logs/v1/logs.proto --gogofaster_out=plugins=grpc:./vendor
	protoc -I opentelemetry-proto/ opentelemetry-proto/opentelemetry/proto/metrics/v1/metrics.proto --gogofaster_out=plugins=grpc:./vendor
	protoc -I opentelemetry-proto/ opentelemetry-proto/opentelemetry/proto/trace/v1/trace.proto --gogofaster_out=plugins=grpc:./vendor
	# protoc -I opentelemetry-proto/ opentelemetry-proto/opentelemetry/proto/collector/logs/v1/logs_service.proto --gogofaster_out=plugins=grpc:./vendor
	protoc -I opentelemetry-proto/ opentelemetry-proto/opentelemetry/proto/collector/metrics/v1/metrics_service.proto --gogofaster_out=plugins=grpc:./vendor
	protoc -I opentelemetry-proto/ opentelemetry-proto/opentelemetry/proto/collector/metrics/v1/metrics_service.proto \
	  --grpc-gateway_out=logtostderr=true,grpc_api_configuration=opentelemetry-proto/opentelemetry/proto/collector/metrics/v1/metrics_service_http.yaml:./vendor
	protoc -I opentelemetry-proto/ opentelemetry-proto/opentelemetry/proto/collector/trace/v1/trace_service.proto --gogofaster_out=plugins=grpc:./vendor
	protoc -I opentelemetry-proto/ opentelemetry-proto/opentelemetry/proto/collector/trace/v1/trace_service.proto \
	  --grpc-gateway_out=logtostderr=true,grpc_api_configuration=opentelemetry-proto/opentelemetry/proto/collector/trace/v1/trace_service_http.yaml:./vendor
	protoc -I opentelemetry-proto/ -I pkg/tempopb/ pkg/tempopb/tempo.proto --gogofaster_out=plugins=grpc:pkg/tempopb

.PHONY: vendor-dependencies
vendor-dependencies:
	go mod vendor
	go mod tidy
	# ignore log.go b/c the proto version used by v0.6.1 doesn't actually have logs proto.
	find . | grep 'vendor/go.opentelemetry.io.*go$\' | grep -v -e 'log.go$\' | xargs -L 1 sed -i $(SED_OPTS) 's+go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/+github.com/open-telemetry/opentelemetry-proto/gen/go/+g'
	$(MAKE) gen-proto


.PHONY: install-tools
install-tools:
	go get -u github.com/golang/protobuf/protoc-gen-go
	go get -u github.com/gogo/protobuf/protoc-gen-gogofaster
	go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway

.PHONE: clear-protos
clear-protos:
	rm -rf opentelemetry-proto

### Check vendored files
.PHONY: vendor-check
vendor-check: clear-protos install-tools vendor-dependencies
	git diff --exit-code

### Release (intended to be used in the .github/workflows/images.yml)
$(GORELEASER):
	curl -sfL https://install.goreleaser.com/github.com/goreleaser/goreleaser.sh | BINDIR=$(GOPATH)/bin sh

release: $(GORELEASER)
	$(GORELEASER) build --skip-validate --rm-dist
	$(GORELEASER) release --rm-dist

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
