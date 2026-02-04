---
name: update-go-version
description: Update Go version across the Tempo codebase (go.mod, tools/go.mod, Dockerfile, CI workflows, tools image tag)
---

# Update Go Version

Updates the Go version across all relevant files in the Tempo codebase.

## Usage

Invoke with `/update-go-version <version>` (e.g., `/update-go-version 1.25.7`)

## Steps to Perform

### 1. Update go.mod files

Update the `go X.Y.Z` directive in both:
- `go.mod` (main module)
- `tools/go.mod` (tools module)

### 2. Update tools/Dockerfile

Update the base image to the new Go version with its SHA256 hash:

```dockerfile
FROM golang:<version>-alpine@sha256:<hash>
```

Get the correct SHA256 hash:
```bash
docker pull golang:<version>-alpine
docker inspect --format='{{index .RepoDigests 0}}' golang:<version>-alpine
```

### 3. Update TOOLS_IMAGE_TAG in build/tools.mk

Fetch the latest tools image tag from Docker Hub:
```bash
curl -s "https://hub.docker.com/v2/repositories/grafana/tempo-ci-tools/tags?page_size=5&ordering=last_updated" | jq -r '.results[0].name'
```

Update `TOOLS_IMAGE_TAG ?= main-XXXXXXX` with the latest tag.

### 4. Verify CI workflow configuration

Check `.github/workflows/ci.yml` uses `go-version-file: 'go.mod'` instead of hardcoded `go-version:` values. Update if needed.

### 5. Verify changes compile

```bash
go mod tidy && go build ./...
cd tools && go mod tidy
```

## Files to Update

| File | What to change |
|------|----------------|
| `go.mod` | `go X.Y.Z` directive |
| `tools/go.mod` | `go X.Y.Z` directive |
| `tools/Dockerfile` | `golang:X.Y.Z-alpine@sha256:...` |
| `build/tools.mk` | `TOOLS_IMAGE_TAG` value |
| `.github/workflows/ci.yml` | Ensure `go-version-file: 'go.mod'` |
