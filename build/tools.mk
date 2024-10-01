#
# Makefile fragment for installing tools
#

GO               ?= go
GOFMT            ?= gofmt
VENDOR_CMD       ?= ${GO} mod tidy
GO_MOD_OUTDATED  ?= go-mod-outdated

# Go file to track tool deps with go modules
TOOL_DIR     ?= tools
TOOL_CONFIG  ?= $(TOOL_DIR)/tools.go

TOOLS_IMAGE ?= grafana/tempo-ci-tools
TOOLS_IMAGE_TAG ?= main-37cd120

GOTOOLS ?= $(shell cd $(TOOL_DIR) && go list -e -f '{{ .Imports }}' -tags tools |tr -d '[]')

TOOLS_CMD = docker run --rm -t -v ${PWD}:/tools $(TOOLS_IMAGE):$(TOOLS_IMAGE_TAG)

.PHONY: tools-image-build
tools-image-build:
	@echo "=== [ tools-image-build]: Building tools image..."
	@docker build -t $(TOOLS_IMAGE) -f ./tools/Dockerfile .

.PHONY: tools-docker
tools-docker:
	@echo "=== [ tools-docker     ]: Running tools in docker..."
	@docker run -it -v $(shell pwd):/var/tempo $(TOOLS_IMAGE_NAME) make -C /var/tempo tools

tools-image:
	@echo "=== [ tools-image     ]: Running tools in docker..."
	@docker pull $(TOOLS_IMAGE):$(TOOLS_IMAGE_TAG)

tools:
	@echo "=== [ tools            ]: Installing tools required by the project..."
	@cd $(TOOL_DIR) && $(GO) install $(GOTOOLS)
	@cd $(TOOL_DIR) && $(VENDOR_CMD)

tools-outdated:
	@echo "=== [ tools-outdated   ]: Finding outdated tool deps with $(GO_MOD_OUTDATED)..."
	@cd $(TOOL_DIR) && $(GO) list -u -m -json all | $(GO_MOD_OUTDATED) -direct -update

tools-update:
	@echo "=== [ tools-update     ]: Updating tools required by the project..."
	@cd $(TOOL_DIR) && for x in $(GOTOOLS); do \
		$(GO) get -u $$x; \
	done
	@cd $(TOOL_DIR) && $(VENDOR_CMD)

.PHONY: tools tools-update tools-outdated

.PHONY: tools-update-mod
tools-update-mod:
	@cd $(TOOL_DIR) && $(VENDOR_CMD)
