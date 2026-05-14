# Tempo — Agent Guidance

## Coding Standards

Before writing or modifying Go code, read [`.agents/guidance/coding.md`](.agents/guidance/coding.md).

## Code Review Standards

Before reviewing code, read [`.agents/guidance/code-review.md`](.agents/guidance/code-review.md).

## Pre-Commit Checklist

Before pushing or opening a PR, read [`.agents/guidance/precommit.md`](.agents/guidance/precommit.md).

## Cursor Cloud specific instructions

### Environment

The VM has Go 1.26.2, golangci-lint v2.11.3, gotestsum, and jsonnetfmt pre-installed. Dependencies are vendored.

### Build, Test, Lint

See `make help` for all targets. Key commands:

- `make tempo` — build the tempo binary to `./bin/linux/tempo-amd64`
- `make test` — run unit tests (uses gotestsum)
- `golangci-lint run --config .golangci.yml` — lint locally (the `make lint` target uses Docker)

### Running Tempo locally

Create `/var/tempo` (writable by agent user) and use a config file with `backend: local`. Example config at `example/docker-compose/single-binary/tempo.yaml` (adjust listen addresses to `0.0.0.0`). The binary accepts `-config.file=<path>`.

### Gotchas

- Docker is not available. Targets that use Docker (e.g., `make fmt`, `make lint`, `make test-e2e`, `make jsonnet`) will fail. Run `golangci-lint` and `gofumpt`/`goimports` directly instead.
- Some `./modules/` tests (Kafka-related) require a running Kafka broker and will fail without Docker. These are infrastructure-dependent, not code bugs.
- The `make test` target uses `gotestsum` — make sure it's installed.
