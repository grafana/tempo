---
name: update-go-version
description: Update Go version across the Tempo codebase (go.mod, tools/go.mod, Dockerfile, CI workflows, tools image tag)
---

# Update Go Version

Updates the Go version across all relevant files in the Tempo codebase.

## Usage

Invoke with `/update-go-version`

## Steps to Perform

### 1. Get the version

Extract Go version from tools/Dockerfile.

This file is updated by a Renovate workflow automatically. If the version was not updated (go.mod has same version), ask user if you should update this file. If user agrees, advise that this change needs to be merged first and new tools image must be built before update can continue.

### 2. Update go.mod files

Update the `go X.Y.Z` directive in both:
- `go.mod` (main module)
- `tools/go.mod` (tools module)

### 3. Update TOOLS_IMAGE_TAG in build/tools.mk

Fetch the latest tools image tag from Docker Hub:
```bash
curl -s "https://hub.docker.com/v2/repositories/grafana/tempo-ci-tools/tags?page_size=5&ordering=last_updated" | jq -r '.results[0].name'
```

Update `TOOLS_IMAGE_TAG ?= main-XXXXXXX` with the latest tag.

### 4. Verify changes compile

```bash
make vendor
make build
```

## Files to Update

| File | What to change |
|------|----------------|
| `go.mod` | `go X.Y.Z` directive |
| `tools/go.mod` | `go X.Y.Z` directive |
| `build/tools.mk` | `TOOLS_IMAGE_TAG` value |
