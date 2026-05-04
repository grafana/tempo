# Metrics Generator Domain Knowledge

Domain-specific knowledge about the Tempo metrics-generator. Useful for both coding and documentation agents.

## Feature Scope

### Span-Metrics Specific Features

These features are **only** available for the `span-metrics` processor:

- `dimension_mappings` - Rename or combine span attributes into custom labels
- `intrinsic_dimensions` - Control which default dimensions are included (service, span_name, span_kind, status_code, status_message)
- `target_info_excluded_dimensions` - Exclude specific attributes from `traces_target_info` metric
- `enable_target_info` - Enable `traces_target_info` metric
- `enable_instance_label` - Control instance label inclusion
- Subprocessors: `span-metrics-latency`, `span-metrics-count`, `span-metrics-size`

**Code Location**: `modules/generator/processor/spanmetrics/`

### Shared Features

These features are available for **multiple processors**:

- `dimensions` - Available in both `span-metrics` and `service-graphs`
- `histogram_buckets` - Available in both processors
- `span_multiplier_key` - Available in both processors
- `filter_policies` - Available in both `span-metrics` and `service-graphs`

**Code Locations**: 
- `modules/generator/processor/spanmetrics/config.go`
- `modules/generator/processor/servicegraphs/config.go`

## Configuration Structure

### Span Metrics Configuration

```yaml
metrics_generator:
  processor:
    span_metrics:
      # Span-metrics specific options
      intrinsic_dimensions:
        service: true
        span_name: true
        span_kind: true
        status_code: true
        status_message: false
      dimension_mappings: []
      enable_target_info: false
      target_info_excluded_dimensions: []
      enable_instance_label: true
      
      # Shared options
      dimensions: []
      histogram_buckets: []
      span_multiplier_key: ""
      filter_policies: []
```

### Service Graphs Configuration

```yaml
metrics_generator:
  processor:
    service_graphs:
      # Service-graphs specific options
      enable_client_server_prefix: false
      enable_messaging_system_latency_histogram: false
      peer_attributes: []
      
      # Shared options
      dimensions: []
      histogram_buckets: []
      span_multiplier_key: ""
      filter_policies: []
```

## Common User Confusion Points

### Issue #3376: dimension_mappings

**Problem**: Users confused about `dimension_mappings` configuration
- Expected `deployment.environment` → `env` but got `deployment_environment`
- Unclear whether to use `dimensions` and `dimension_mappings` together
- Confused about `source_labels` format (dots vs. underscores)

**Solution**:
- Explicitly state `source_labels` must use original attribute names (with dots)
- Explain that `dimensions` and `dimension_mappings` are complementary: you can use either one alone or combine them, but after label sanitization the final label names must not collide across intrinsic dimensions, `dimensions`, and `dimension_mappings`
- Provide a clear example that shows both actual renaming and how sanitized label names differ from the original attributes, including how collisions are avoided

**Key Insight**: `dimension_mappings` reads directly from original span attributes, not from `dimensions` output.

## Default Labels

### Span Metrics Default Labels

**Always included by default**:
- `service` - Service name
- `span_name` - Span name
- `span_kind` - Span kind (SERVER, CLIENT, INTERNAL, PUBLISHER, CONSUMER)
- `status_code` - Status code (UNSET, OK, ERROR)

**Optional labels** (require configuration):
- `status_message` - Must enable via `intrinsic_dimensions.status_message: true`
- `job` - Only if `enable_target_info: true`
- `instance` - Only if `enable_target_info: true` AND `enable_instance_label: true`

**Code Reference**: `modules/generator/processor/spanmetrics/config.go:68-71`

## Metrics Generated

### Span Metrics

Three metrics are generated:
1. `traces_spanmetrics_latency` - Histogram (duration)
2. `traces_spanmetrics_calls_total` - Counter (requests)
3. `traces_spanmetrics_size_total` - Counter (span size)

**Code Reference**: `modules/generator/processor/spanmetrics/spanmetrics.go:24-27`

### Subprocessors

Users can enable individual metrics via subprocessors:
- `span-metrics-latency` → only `traces_spanmetrics_latency`
- `span-metrics-count` → only `traces_spanmetrics_calls_total`
- `span-metrics-size` → only `traces_spanmetrics_size_total`

**Code Reference**: `modules/generator/processor/processor_names.go:6-8`

## Version Compatibility

### Tempo 2.10 Features

All span-metrics features documented are available in Tempo 2.10:
- `dimension_mappings` - Exposed to user-configurable overrides API in 2.10 (PR #5989)
- `intrinsic_dimensions` - Exposed to user-configurable overrides API in 2.10 (PR #5974)
- `target_info_excluded_dimensions` - Added in 2.3 (PR #2945)
- `enable_instance_label` - Added in 2.10 (PR #5706)
- Subprocessors - Bug fix in 2.5 (PR #3612), feature existed earlier

**Note**: The span-metrics and service-graphs processor features listed above are **not** new in Tempo 3.0. However, the metrics-generator's architecture changed significantly in 3.0 (see below).

### Tempo 3.0 Architectural Changes

Tempo 3.0 (Project Rhythm) restructures how the metrics-generator receives data and how recent-data queries are served. The span-metrics and service-graphs processors themselves are unchanged, but the surrounding plumbing is different.

| Aspect | v2.x | v3.0 |
|--------|------|------|
| **Data source** | Distributor pushes spans to the metrics-generator via gRPC (`PushSpans`) | Distributor writes spans to Kafka (microservices mode) or calls `PushSpans` in-process (single-binary; no generator gRPC ingestion service) |
| **local-blocks processor** | Supported | **Removed** (PR #6555) |
| **Recent-data metrics queries** | Served by metrics-generator / ingesters | Served by **live-store** |
| **SpanMetricsSummary API** | Supported | **Removed** (PRs #6496, #6510) |
| **gRPC server** | Always on for ingestion (`PushSpans`) and query APIs | Used only for query APIs; distributor never calls generator gRPC in v3.0 |
| **Valid processors** | service-graphs, span-metrics, local-blocks, host-info | service-graphs, span-metrics, host-info |

#### Kafka ingestion

In Tempo 3.0 microservices mode, the metrics-generator consumes trace data from Kafka. In single-binary mode, the Tempo process calls the generator's `PushSpans` method directly in-process, without a generator gRPC ingestion service.

- **Implementation**: `modules/generator/generator_kafka.go`
- **Config**: top-level `ingest` block — address, topic, consumer group, etc. (injected into the generator at runtime; the generator config uses `yaml:"-"` so it doesn't appear under `metrics_generator`)
- **Codec**: `push-bytes` (default) or `otlp` (`modules/generator/config.go`)
- **Concurrency**: `ingest_concurrency` (default 16)

#### Two generator modes

- **`MetricsGenerator`** (traditional) — Uses ring + memberlist. Can receive PushSpans and/or consume Kafka. gRPC optional.
- **`MetricsGeneratorNoLocalBlocks`** — Kafka-only mode. Uses partition ring watcher and consumes from Kafka as configured by the deployment model. (`disable_grpc` is deprecated and ignored.)

**Code Reference**: `cmd/tempo/app/modules.go`

#### Removed: local-blocks processor

The `local-blocks` processor and all local block storage plumbing are removed in v3. The live-store now fills this role for TraceQL metrics queries on recent data.

**Breaking change**: Any configuration referencing `local-blocks` as a processor must be removed.

#### Removed: SpanMetricsSummary

The `SpanMetricsSummary` API and related querier logic are removed.

#### Live-store decoupling

- Recent-data TraceQL metrics queries are now served by the live-store, **not** the metrics-generator (PRs #6506, #6535, #6615).
- The querier uses `forLiveStoreMetricsRing` to query the live-store's `MetricsClient`.
- `query_frontend.search.query_ingesters_until` is removed; only `query_backend_after` remains (PR #6507).

#### New Kafka-related operational metrics

- `tempo_ingest_group_partition_lag{group="metrics-generator"}`
- `tempo_ingest_group_partition_lag_seconds{group="metrics-generator"}`
- `tempo_metrics_generator_enqueue_time_seconds_total`

## Configuration Reference Locations

- Main config reference: `docs/sources/tempo/configuration/_index.md`
- Operations overrides: `docs/sources/tempo/operations/manage-advanced-systems/user-configurable-overrides.md`

## Key Code Files

### Span Metrics Implementation
- `modules/generator/processor/spanmetrics/spanmetrics.go` - Main processor logic
- `modules/generator/processor/spanmetrics/config.go` - Configuration struct
- `modules/generator/processor/spanmetrics/subprocessors.go` - Subprocessor definitions
- `modules/generator/processor/processor_names.go` - Processor name constants
- `modules/generator/processor/dimension_names.go` - Dimension name constants

### Generator Core (v3)
- `modules/generator/generator.go` - Main generator logic, `startKafka()`
- `modules/generator/generator_kafka.go` - Kafka consumer implementation
- `modules/generator/config.go` - Config struct including `Ingest`, `Codec`, `DisableGRPC` (deprecated)
- `pkg/ingest/encoding.go` - `PushBytesDecoder` and `OTLPDecoder` for Kafka records
- `cmd/tempo/app/modules.go` - Module registration for both generator modes

### Validation
- `modules/generator/validation/fields.go` - Valid processor names
- `pkg/sharedconfig/metrics_generator.go` - Shared config structs

## Common Patterns

### dimension_mappings Pattern

```yaml
dimension_mappings:
  - name: env                    # Custom label name
    source_labels: ["deployment.environment"]  # Original attribute name (with dots!)
```

**Critical**: `source_labels` must use original attribute names with dots, not sanitized names.

### intrinsic_dimensions Pattern

```yaml
intrinsic_dimensions:
  service: true       # Default: true
  span_name: true     # Default: true
  span_kind: false    # Disable to reduce cardinality
  status_code: true   # Default: true
  status_message: false  # Default: false
```
