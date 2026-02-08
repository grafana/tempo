# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is Tempo

Grafana Tempo is a high-scale distributed tracing backend. It ingests traces (Jaeger, OpenTelemetry, Zipkin formats), stores them in object storage (S3, GCS, Azure) using Parquet, and supports trace lookup by ID and TraceQL search queries.

## Build Commands

```bash
make tempo              # Build main binary
make tempo-cli          # Build CLI tool
make tempo-query        # Build Jaeger query plugin
make tempo-vulture      # Build consistency checker
```

Binaries output to `./bin/<GOOS>/tempo-<GOARCH>`. CGO is disabled. Uses `-mod vendor`.

## Testing

```bash
# Run all unit tests
make test

# Run a single test
gotestsum --format=testname -- -race -timeout 25m -count=1 -v -run TestName ./pkg/some/package/

# CI splits tests into groups:
make test-with-cover-pkg           # pkg/ tests
make test-with-cover-tempodb       # tempodb/ tests (uses GOMEMLIMIT=6GiB)
make test-with-cover-tempodb-wal   # tempodb/wal tests
make test-with-cover-others        # everything else

# E2E tests (require docker images built first)
make test-e2e-operations
make test-e2e-api
make test-e2e-limits
make test-e2e-metrics-generator
make test-e2e-storage
```

Test flags: `-race -timeout 25m -count=1 -v`

## Linting and Formatting

```bash
make lint           # golangci-lint (config: .golangci.yml)
make fmt            # gofumpt + goimports (runs in tools Docker image)
make check-fmt      # fmt then check for diffs
make jsonnetfmt     # format jsonnet/libsonnet files
```

Linters enabled: errorlint, forbidigo, gocritic, gosec, goconst, misspell, revive, unconvert, unparam.

Files excluded from formatting: `*.pb.go`, `*.y.go`, `*.gen.go`, vendor/.

## Code Generation

```bash
make gen-proto          # Protobuf (runs in Docker with Buf)
make gen-traceql        # TraceQL parser from yacc grammar (pkg/traceql/expr.y)
make gen-parquet-query  # Parquet query predicates
make vendor-check       # Verify vendored deps + generated code are in sync
```

After changing dependencies: `go mod tidy -e && go mod vendor`, then `make vendor-check`.

## Architecture

Tempo uses a microservices architecture (can also run as single binary). Components communicate via gRPC and are coordinated through hash rings (memberlist).

### Core Components (modules/)

- **distributor** - Accepts spans via OTel/Jaeger/Zipkin receivers, hashes trace IDs, routes to ingesters via consistent hash ring
- **ingester** - Buffers incoming spans, indexes attributes into Parquet schema, builds blocks, flushes to object storage
- **querier** - Queries both ingesters (recent data) and object storage (historical data)
- **frontend** - Query entry point; shards and parallelizes TraceQL and trace-by-ID queries across queriers
- **generator** - Optional; derives metrics from ingested traces, writes to metrics storage
- **storage** - Abstraction layer over tempodb
- **overrides** - Per-tenant runtime limits and configuration
- **backendscheduler / backendworker** - Job scheduling for compaction and retention (replaces the compactor)
- **blockbuilder** - Builds storage blocks
- **livestore** - Live trace storage with ring-based coordination

### Storage Layer (tempodb/)

Key-value database on object storage. Sub-packages:
- **backend/** - Storage backends: S3, GCS, Azure, local filesystem
- **encoding/** - Block encoding formats (current version: `vparquet5`)
- **wal/** - Write-ahead log
- **blocklist/** - Block metadata management

### Parquet Schema and Attribute Storage

The Parquet schema is defined via Go struct tags in `tempodb/encoding/vparquet5/schema.go`. Traces are stored as `Trace > ResourceSpans > ScopeSpans > Span`, mirroring the OTel data model.

All three OTel attribute scopes are stored in Parquet but with different optimization levels:

**Resource and Span attributes** use a three-tier storage strategy, applied in priority order by `traceToParquetWithMapping()`:

1. **`service.name` gets its own column** on `Resource.ServiceName` — extracted as a top-level field for fast filtering since it's the most common search predicate.
2. **Dedicated columns** — pre-allocated spare columns (`DedicatedAttributes.String01`–`String20`, `Int01`–`Int05`) for frequently-queried attributes, configured per block. Both `Resource` and `Span` have their own `DedicatedAttributes`. Avoids scanning generic attribute arrays.
3. **Generic `Attrs[]`** — all remaining attributes stored as `[]Attribute`, where each `Attribute` has a `Key` and polymorphic typed value fields (`Value []string`, `ValueInt []int64`, `ValueDouble []float64`, `ValueBool []bool`). Exactly one type field is populated per attribute. Unsupported types (nested KV lists, byte arrays) fall back to JSON in `ValueUnsupported`.

**InstrumentationScope attributes** (on `ScopeSpans.Scope`) use only the generic `Attrs[]` — no dedicated columns or special extraction. The scope's `Name` and `Version` are stored as their own dictionary-encoded string columns.

On read, `ParquetTraceToTempopbTrace()` merges all tiers back into `[]KeyValue` to reconstruct the original OTel attributes. All string columns use Snappy compression with dictionary encoding.

### Shared Packages (pkg/)

- **tempopb/** - Protobuf definitions (patched OTel protos with `tempopb` package namespace)
- **traceql/** - TraceQL query language parser and engine (yacc-generated from `expr.y`)
- **parquetquery/** - Parquet query layer
- **api/** - HTTP API handlers
- **model/** - Data models and trace representations

### Entry Points (cmd/)

- **tempo** - Main server; uses dskit module manager pattern for component lifecycle. Config loaded from YAML with env var expansion.
- **tempo-cli** - Kong-based CLI for backend inspection (list blocks, query traces, analyze parquet)
- **tempo-query** - Jaeger gRPC query plugin
- **tempo-vulture** - Writes and reads traces to verify system consistency

### Integration Tests (integration/)

Docker-based E2E tests organized by feature area. Test harness in `integration/util/` manages component orchestration and provides helpers for writing/reading traces.

## Coding Conventions

- **Imports**: Three groups separated by blank lines: std lib, external, local (`github.com/grafana/tempo/...`)
- **Formatting**: gofumpt (not just gofmt)
- **Logging**: go-kit level logging, key=value (logfmt) format
- **Metrics**: Prometheus (RED pattern); dashboards in `operations/tempo-mixin`
- **Tracing**: OpenTelemetry instrumentation
- **Dependencies**: Vendored (`-mod vendor`). Run `make vendor-check` before submitting PRs
- **Proto files**: Patched OTel protos under `tempopb` namespace to avoid conflicts with downstream OTel usage

## Docker Images

```bash
make docker-tempo          # Build tempo image
make docker-tempo-query    # Build tempo-query image
make docker-tempo-vulture  # Build vulture image
make docker-tempo-cli      # Build CLI image
```

## Documentation

```bash
make docs       # Preview docs at localhost:3002 (uses grafana/docs-base image)
make docs-test  # Build docs for validation
```

Docs source is in `docs/sources/tempo/`. Design proposals in `docs/design-proposals/` (not published).
