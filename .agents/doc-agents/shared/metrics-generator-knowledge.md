# Metrics Generator Domain Knowledge

This document captures domain-specific knowledge about the Tempo metrics-generator to help with accurate documentation.

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
- `filter_policies` - Available in span-metrics

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
```

## Common User Confusion Points

### Issue #3376: dimension_mappings

**Problem**: Users confused about `dimension_mappings` configuration
- Expected `deployment.environment` → `env` but got `deployment_environment`
- Unclear whether to use `dimensions` and `dimension_mappings` together
- Confused about `source_labels` format (dots vs. underscores)

**Solution**:
- Explicitly state `source_labels` must use original attribute names (with dots)
- Explain `dimensions` and `dimension_mappings` are alternatives, not complementary
- Provide clear example showing actual renaming, not default sanitization

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

**Note**: These are **not** part of Project Rhythm/Tempo 3.0. Project Rhythm focuses on architectural changes (Kafka, live stores, block builders), not metrics-generator features.

## Configuration Reference Locations

- Main config reference: `docs/sources/tempo/configuration/_index.md` (lines 496-549 for span_metrics)
- User-configurable overrides: `docs/sources/tempo/configuration/_index.md` (lines 2033-2053)
- Operations overrides: `docs/sources/tempo/operations/manage-advanced-systems/user-configurable-overrides.md`

## Key Code Files

### Span Metrics Implementation
- `modules/generator/processor/spanmetrics/spanmetrics.go` - Main processor logic
- `modules/generator/processor/spanmetrics/config.go` - Configuration struct
- `modules/generator/processor/spanmetrics/subprocessors.go` - Subprocessor definitions
- `modules/generator/processor/processor_names.go` - Processor name constants
- `modules/generator/processor/dimension_names.go` - Dimension name constants

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

## Documentation Gaps Found

### Missing from Config Reference

The configuration reference (`_index.md` line 1964-1967) only lists:
- `service-graphs`
- `span-metrics`
- `local-blocks`

But code supports subprocessors:
- `span-metrics-latency`
- `span-metrics-count`
- `span-metrics-size`
- `host-info`

**Note**: This is a gap in the config reference, not the span-metrics documentation.

## Testing Documentation

When documenting metrics-generator features:

1. Verify against `modules/generator/processor/spanmetrics/` code
2. Check `CHANGELOG.md` for introduction version
3. Compare with `docs/sources/tempo/configuration/_index.md`
4. Test examples against actual configuration structure
5. Verify feature scope (span-metrics only vs. shared)
