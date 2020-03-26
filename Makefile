# More exclusions can be added similar with: -not -path './testbed/*'
ALL_SRC := $(shell find . -name '*.go' \
                                -not -path './testbed/*' \
								-not -path './vendor/*' \
                                -type f | sort)

# All source code and documents. Used in spell check.
ALL_DOC := $(shell find . \( -name "*.md" -o -name "*.yaml" \) \
                                -type f | sort)

# ALL_PKGS is used with 'go cover'
ALL_PKGS := $(shell go list $(sort $(dir $(ALL_SRC))))

GO_OPT= -mod vendor
GOTEST_OPT?= -race -timeout 30s -count=1
GOTEST_OPT_WITH_COVERAGE = $(GOTEST_OPT) -cover
GOTEST=go test
LINT=golangci-lint

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
	GO111MODULE=on CGO_ENABLED=0 go build $(GO_OPT) -o ./bin/$(GOOS)/tempo-vulture $(BUILD_INFO) ./cmd/tempo-cli

.PHONY: test
test:
	$(GOTEST) $(GOTEST_OPT) $(ALL_PKGS)

.PHONY: benchmark
benchmark:
	$(GOTEST) -bench=. -run=notests $(ALL_PKGS)

.PHONY: test-with-cover
test-with-cover:
	$(GOTEST) $(GOTEST_OPT_WITH_COVERAGE) $(ALL_PKGS)

.PHONY: lint
lint:
	$(LINT) run

.PHONY: docker-component # Not intended to be used directly
docker-component: check-component
	GOOS=linux $(MAKE) $(COMPONENT)
	cp ./bin/linux/$(COMPONENT) ./cmd/$(COMPONENT)/
	docker build -t $(COMPONENT) ./cmd/$(COMPONENT)/
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

.PHONY: check-component
check-component:
ifndef COMPONENT
	$(error COMPONENT variable was not defined)
endif

.PHONY: gen-proto
gen-proto:
	vend -package
	protoc -I vendor/github.com/open-telemetry/opentelemetry-proto -I pkg/tempopb/ pkg/tempopb/tempo.proto --go_out=plugins=grpc:pkg/tempopb
	$(MAKE) vendor-dependencies

.PHONY: vendor-dependencies
vendor-dependencies:
	go mod tidy
	go mod vendor

.PHONY: install-tools
install-tools:
	go get -u github.com/nomad-software/vend
	go get -u github.com/golang/protobuf/protoc-gen-go
