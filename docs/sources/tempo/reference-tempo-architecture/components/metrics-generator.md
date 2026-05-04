---
title: Metrics-generator
menuTitle: Metrics-generator
description: How the metrics-generator derives metrics from trace data.
weight: 800
topicType: concept
versionDate: 2026-03-20
---

# Metrics-generator

The metrics-generator is an optional component that derives metrics from trace data,
which are then remote-written to a metrics backend, for example, Prometheus or Grafana Mimir.

How the metrics-generator receives data depends on the [deployment mode](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/deployment-modes/):

- Microservices mode: The metrics-generator consumes trace data from Kafka as an independent consumer group.
- Monolithic mode: The metrics-generator receives trace data directly from the distributor in-process. No Kafka consumption is involved.

## Why it matters

Traces contain rich information about service interactions, latencies, and error rates.
The metrics-generator extracts this information and produces time-series metrics,
enabling alerting and Grafana dashboards without requiring separate instrumentation.

It supports two types of metric generation.
Span metrics produce request rate, error rate, and duration (RED) metrics from individual spans.
These can be broken down by service, operation, status code, and custom dimensions extracted from span attributes.
Service graphs build a graph of service-to-service communication by matching client and server spans,
producing metrics for request rates, error rates, and latencies between service pairs.

## Kafka consumption

In microservices mode, the metrics-generator consumes trace data directly from Kafka, like live-stores and block-builders.
It runs as an independent consumer group, tracking its own offsets separately.

### Monitoring consumption

Use the following metrics to verify the generator is consuming data:

```
tempo_ingest_group_partition_lag{group="metrics-generator"}
tempo_ingest_group_partition_lag_seconds{group="metrics-generator"}
```

High or growing lag indicates the generator is falling behind.
The `tempo_ingest_storage_reader` family of metrics exposes detailed information about fetch operations and errors from the Kafka client library.

## Active series limiting

The generator protects itself and downstream metrics storage with configurable limits.

### Series-based limiting

You can cap the total number of active time series the generator produces:

```yaml
overrides:
  defaults:
    metrics_generator:
      max_active_series: 0  # 0 = unlimited
```

This value is per metrics-generator instance. The actual maximum across the cluster is `<instances> * max_active_series`.

When the limit is reached, the generator produces overflow series with the label `metric_overflow="true"` instead of dropping data entirely.
As existing series become stale, new series split out from the overflow bucket.

### Entity-based limiting

Entity-based limiting is an alternative to series-based limiting.
An entity is a unique label combination (excluding external labels) across multiple metrics.
Entity limiting ensures the generator always produces the full set of metrics for a given entity rather than limiting randomly.

```yaml
metrics_generator:
  limiter_type: entity
```

### Per-label cardinality limiting

You can cap the number of distinct values a single label can have.
When exceeded, new values are replaced with `__cardinality_overflow__` while other labels remain unaffected.

```yaml
overrides:
  defaults:
    metrics_generator:
      max_cardinality_per_label: 0  # 0 = disabled
```

## Remote write

The generator writes metrics to one or more remote-write endpoints. Monitor write health with:

```
prometheus_remote_storage_samples_failed_total
prometheus_remote_storage_samples_dropped_total
```

## Related resources

Refer to the [metrics-generator documentation](https://grafana.com/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/metrics-generator/) for configuration and usage details.
