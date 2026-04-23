#
# Makefile fragment for installing tools
#

GO               ?= go
GOFMT            ?= gofmt
VENDOR_CMD       ?= ${GO} mod tidy
GO_MOD_OUTDATED  ?= go-mod-outdated

TOOL_DIR     ?= tools

TOOLS_IMAGE ?= grafana/tempo-ci-tools
TOOLS_IMAGE_TAG ?= main-2c55cd0-20260421-172344

# Mount the git common directory to the tools container.
# This is needed when using git worktrees.
GIT_COMMON_DIR := $(shell git rev-parse --git-common-dir 2>/dev/null)
ifneq ($(strip $(GIT_COMMON_DIR)),)
  ifneq ($(GIT_COMMON_DIR),.git)
    WORKTREE_DOCKER_MOUNT := -v $(GIT_COMMON_DIR):$(GIT_COMMON_DIR)
  endif
endif

TOOLS_CMD = docker run --rm -t -v ${PWD}:/tools $(WORKTREE_DOCKER_MOUNT) $(TOOLS_IMAGE):$(TOOLS_IMAGE_TAG)
LINT_CMD =  docker run --rm -t -v ${PWD}:/tools $(WORKTREE_DOCKER_MOUNT) -v ${PWD}/.cache/golangci-lint:/root/.cache/golangci-lint $(TOOLS_IMAGE):$(TOOLS_IMAGE_TAG)

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
	@cd $(TOOL_DIR) && $(GO) install tool
	@cd $(TOOL_DIR) && $(VENDOR_CMD)

tools-outdated:
	@echo "=== [ tools-outdated   ]: Finding outdated tool deps with $(GO_MOD_OUTDATED)..."
	@cd $(TOOL_DIR) && $(GO) list -u -m -json all | $(GO_MOD_OUTDATED) -update

tools-update:
	@echo "=== [ tools-update     ]: Updating tools required by the project..."
	@cd $(TOOL_DIR) && $(GO) get -u tool
	@cd $(TOOL_DIR) && $(VENDOR_CMD)

.PHONY: tools tools-update tools-outdated

.PHONY: tools-update-mod
tools-update-mod:
	@cd $(TOOL_DIR) && $(VENDOR_CMD)
