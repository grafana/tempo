---
name: update-go-version
description: Update Go version across the Tempo codebase (go.mod, tools/go.mod, Dockerfile, CI workflows, tools image tag)
allowed-tools: WebFetch, Grep, Read, Write
---

# Update Go Version

Updates the Go version across all relevant files in the Tempo codebase.

## Usage

Invoke with `/update-go-version`

## Steps to Perform

### 1. Get the version

Extract Go version from tools/Dockerfile. 

This file is updated by a Renovate workflow automatically.

### 2. Check if go.mod files need updating

Check these files:
- `go.mod` (main module)
- `tools/go.mod` (tools module)
If the versions already match, advise user that tools/Dockerfile needs to be updated and merged first to build new image, then stop.

### 3. Update go.mod files

Update the `go X.Y.Z` directive in both:
- `go.mod` (main module)
- `tools/go.mod` (tools module)

### 4. Update TOOLS_IMAGE_TAG in build/tools.mk

Fetch the latest tools image tag from Docker Hub:
```bash
curl -s "https://hub.docker.com/v2/repositories/grafana/tempo-ci-tools/tags?page_size=5&ordering=last_updated" | jq -r '.results[0].name'
```

Update `TOOLS_IMAGE_TAG ?= main-XXXXXXX` with the latest tag.

### 5. Verify changes compile

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
