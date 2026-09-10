---
title: Troubleshoot metrics-generator
menuTitle: Metrics-generator
description: Gain an understanding of how to debug metrics quality issues.
weight: 500
aliases:
  - ../operations/troubleshooting/metrics-generator/ # https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/troubleshooting/metrics-generator/
---

# Troubleshoot metrics-generator

If you're concerned with data quality issues in the metrics-generator, consider:

- Reviewing your telemetry pipeline to determine the number of dropped spans. You are only looking for major issues here.
- Reviewing the [service graph documentation](/docs/tempo/<TEMPO_VERSION>/metrics-generator/service_graphs/) to understand how they are built.

If everything seems acceptable from these two perspectives, consider the following topics to help resolve general issues with all metrics and span metrics specifically.

## Common configuration issues

The following issues come from configuration rather than data quality.

### No metrics or only partial metrics after enabling

If you enabled the metrics-generator but no series are created, or the metrics cover only a fraction of your traffic, work through this checklist.
The generator doesn't require any specific span kind or attribute to produce metrics, so a missing label is rarely the cause.

1. **Confirm at least one processor is enabled.**
   Processors are disabled by default.
   Set `metrics_generator.processors` in the [overrides](/docs/tempo/<TEMPO_VERSION>/configuration/#overrides) block, for example `[span-metrics, service-graphs]`.
1. **Confirm the generator is receiving spans.**
   In a Tempo 3.0 microservices deployment, the generator consumes from Kafka.
   Verify consumption using the metrics in [Kafka consumption](#kafka-consumption).
1. **Check for discarded spans.**
   Spans with end times older than `metrics_ingestion_time_range_slack` (default 30 seconds) are discarded.
   Refer to [Discarded spans in the generator](#discarded-spans-in-the-generator).
1. **Confirm remote-write is working.**
   Check `prometheus_remote_storage_samples_failed_total` and refer to [Remote write failures](#remote-write-failures).
1. **Confirm series are being produced at all.**
   `tempo_metrics_generator_registry_active_series` should be greater than zero.
   If it's zero, the generator isn't producing series, which points to processors or ingestion rather than remote-write.
1. **Check your filter policies.**
   An overly strict `include` policy can drop every span.
   Refer to [Filter policies aren't matching spans](#filter-policies-arent-matching-spans).
1. **Account for sampling.**
   The generator only sees the spans that reach it.
   If you sample traces upstream, for example with tail sampling in Grafana Alloy or the OpenTelemetry Collector, the generator produces metrics only for the sampled spans.
   With 10% sampling, expect roughly 10% of the request rate.
   To decide where to generate metrics relative to sampling, refer to [Choose where to generate metrics from traces](/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/where-to-generate-metrics/).

### Unexpected metric types are generated

If you're getting metrics you didn't want, such as `traces_spanmetrics_latency` histograms when you only wanted `traces_spanmetrics_size_total`, the cause is usually processor selection.

Enabling the `span-metrics` processor generates all three span metrics: counts, latency histograms, and sizes.
To generate only specific metrics, enable the individual subprocessors instead of `span-metrics`:

- `span-metrics-count` emits only `traces_spanmetrics_calls_total`.
- `span-metrics-latency` emits only `traces_spanmetrics_latency`.
- `span-metrics-size` emits only `traces_spanmetrics_size_total`.

For example, to generate counts and sizes but no latency histogram:

```yaml
overrides:
  defaults:
    metrics_generator:
      processors:
        - span-metrics-count
        - span-metrics-size
```

For more information, refer to [Enabling specific metrics (subprocessors)](/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/span-metrics/span-metrics-metrics-generator/#enabling-specific-metrics-subprocessors).

### Filter policies aren't matching spans

If a filter policy doesn't include or exclude the spans you expect, check for these common mistakes:

- **Missing attribute scope.**
  Non-intrinsic attributes must be prefixed with their scope, such as `resource.service.name` or `span.http.route`.
  A bare `service.name` doesn't match.
  Intrinsic keys `name`, `status`, and `kind` are used without a scope.
- **Wrong intrinsic value.**
  `kind` values must be the full `SPAN_KIND_*` form, such as `SPAN_KIND_SERVER`.
  `status` values must be the full `STATUS_CODE_*` form.
- **Regex escaping.**
  With `match_type: regex`, the value is a regular expression.
  Escape literal dots, and remember that the pattern isn't implicitly anchored.
- **Unexpected include logic.**
  Multiple `include` policies are combined with logical AND, so a span must match all of them.
  Use `include_any` for logical OR.

To measure how many spans your filters drop, use the discarded-spans metric:

```
sum(rate(tempo_metrics_generator_spans_discarded_total{}[1m])) by (reason)
```

For the full filter policy syntax and worked examples, refer to [Filtering](/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/span-metrics/span-metrics-metrics-generator/#filtering).

### Data gaps after configuration changes

Changing a processor's label set, for example by disabling an intrinsic dimension such as `span_name` or by adding or removing a dimension, retires the existing series and starts new ones.
This can cause a brief gap in metric generation before metrics resume, usually until the next collection interval.

This is expected behavior.
If a short gap would affect alerting, apply label-set changes during a maintenance window.

## Kafka consumption

In Tempo 3.0 microservices mode, metrics-generators consume trace data directly from Kafka rather than receiving pushes from distributors. In monolithic mode, the distributor still pushes directly to the in-process metrics-generator. If the generator is not producing metrics in a microservices deployment, start by verifying that it's consuming data from Kafka using the metrics below.

### Consumer lag

Use the following metrics to monitor the generator's Kafka consumer lag:

```
tempo_ingest_group_partition_lag{group="metrics-generator"}
tempo_ingest_group_partition_lag_seconds{group="metrics-generator"}
```

`tempo_ingest_group_partition_lag` tracks the lag in number of records per partition, while `tempo_ingest_group_partition_lag_seconds` tracks the lag in seconds. High or growing lag indicates the generator is falling behind.

### Kafka client errors

The generator uses the `tempo_ingest_storage_reader` family of metrics (provided by the Kafka client library) to expose detailed information about fetch operations, errors, and throughput. Look for error and failure metrics in this family to diagnose connectivity or protocol issues with Kafka.

## All metrics

This section covers additional metrics related to the metrics-generator.

### Discarded spans in the generator

Spans are rejected from being considered by the metrics-generator by a configurable slack time as well as due to user
configurable filters. You can see the number of spans rejected by reason using this metric:

```
sum(rate(tempo_metrics_generator_spans_discarded_total{}[1m])) by (reason)
```

If a lot of spans are dropped in the metrics-generator due to your filters, you will need to adjust them. If spans are dropped
due to the ingestion slack time, consider adjusting this setting:

```
metrics_generator:
  metrics_ingestion_time_range_slack: 30s
```

If spans are regularly exceeding this value you may want to consider reviewing your tracing pipeline to see if you have excessive buffering.
Note that increasing this value allows the generator to consume more spans, but does reduce the accuracy of metrics because spans farther
away from "now" are included.

Spans could also be discarded if the attributes aren't valid UTF-8 characters when those attributes are converted to metric labels.

### Max active series

The generator protects itself and your remote-write target by having a maximum number of series the generator produces.
Use the `sum` below to determine if series are being dropped due to this limit:

```
sum(rate(tempo_metrics_generator_registry_series_limited_total{}[1m]))
```

Use the following setting to update the limit:

```yaml
overrides:
  defaults:
    metrics_generator:
      max_active_series: 0
```

Note that this value is per metrics generator. The actual max series remote written will be `<# of metrics generators> * <metrics_generator.max_active_series>`.

### Overflow series

When the active series limit is reached, the metrics-generator produces overflow series instead of dropping new data. These series have the label `metric_overflow="true"` and capture all data that would otherwise be lost.

To identify overflow series in your metrics:

```promql
{metric_overflow="true"}
```

As existing series become stale and are removed, new series are split out from the overflow bucket until the limit is reached again. To reduce overflow, either increase `max_active_series` or reduce cardinality by adjusting dimensions or filters.

### Entity-based limiting

You can configure entity-based limiting as an alternative to series-based limiting.
An entity is a unique label combination (excluding external labels) across multiple metrics.
Entity-based limiting ensures the generator always produces the full set of metrics for a given entity, rather than limiting randomly once the series limit is triggered.

To enable entity-based limiting, set `limiter_type` to `entity`:

```yaml
metrics_generator:
  limiter_type: entity
```

Use the following metric to determine if entities are being limited:

```
sum(rate(tempo_metrics_generator_registry_entities_limited_total{}[1m]))
```

Configure the entity limit with:

```yaml
overrides:
  defaults:
    metrics_generator:
      max_active_entities: 0
```

### Per-label cardinality limiting

The per-label cardinality limiter caps the number of distinct values any single label can have. When a label exceeds the configured threshold, its value is replaced with `__cardinality_overflow__` while all other labels that are under the limit are preserved.

For example, if the `url` label exceeds the cardinality limit:

Before:
```
{service="foo", method="GET", url="/users/1"}
{service="foo", method="GET", url="/users/2"}
{service="foo", method="GET", url="/users/3"}
...
```

After:
```
{service="foo", method="GET", url="__cardinality_overflow__"}
```

Once the limiter kicks in, new `url` values are replaced with `__cardinality_overflow__`. Labels that remain under the limit, like `method`, are unaffected.

To detect if per-label cardinality limiting is active:

```promql
sum by (tenant, label_name) (rate(tempo_metrics_generator_registry_label_values_limited_total{}[5m]))
```

To view the estimated cardinality demand per label:

```promql
tempo_metrics_generator_registry_label_cardinality_demand_estimate{}
```

Use this metric to identify which labels have high cardinality, how far they exceed the configured limit, and to choose an appropriate
`max_cardinality_per_label` value. To observe actual demand before enforcing a limit, deploy with a high `max_cardinality_per_label` value first.

#### Understand the `label_name` values in this metric

The `label_name` label values represent every label tracked by the per-label cardinality limiter.
These include all labels that flow through the metrics-generator registry, not just user-configured dimensions.

Built-in labels:

| Label             | Processor      | When added                                                       | Description                                               |
|-------------------|----------------|------------------------------------------------------------------|-----------------------------------------------------------|
| `service`         | span-metrics   | Always                                                           | The service name                                          |
| `span_name`       | span-metrics   | Always                                                           | The operation or span name                                |
| `span_kind`       | span-metrics   | Always                                                           | The span kind (SERVER, CLIENT, etc.)                      |
| `status_code`     | span-metrics   | Always                                                           | The span status code                                      |
| `job`             | span-metrics   | `enable_target_info` is `true`                                   | The job name, derived from resource attributes            |
| `instance`        | span-metrics   | `enable_target_info` and `enable_instance_label` are both `true` | The instance ID, derived from resource attributes         |
| `client`          | service-graphs | Always                                                           | The client service name                                   |
| `server`          | service-graphs | Always                                                           | The server service name                                   |
| `connection_type` | service-graphs | Always                                                           | The connection type (virtual, database, messaging_system) |

Configured labels include:

- Span-metrics dimensions are added as-is.
  For example, `deployment.environment` becomes `deployment_environment`.
- Service-graphs dimensions are prefixed with `client_` and `server_` when `enable_client_server_prefix` is `true`.
  For example, `deployment.environment` becomes `client_deployment_environment` and `server_deployment_environment`.
- A configured dimension only appears if the corresponding attribute exists on incoming spans.

Configure the per-label cardinality limit:

```yaml
overrides:
  defaults:
    metrics_generator:
      max_cardinality_per_label: 0
```

A value of `0` (default) disables the limit.

This setting works alongside both active series limiting (`max_active_series`) and entity-based limiting (`max_active_entities`).
The per-label limiter runs during label construction, preventing any single high-cardinality label from consuming the entire active series or entity budget.

The per-label limiter uses HyperLogLog sketches to estimate cardinality, so the limit is approximate with a 3.25% standard error. Estimates are
re-evaluated every few seconds, which means there may be a brief delay between a label crossing the threshold and the limiter taking effect.

If a high-cardinality label's cardinality is later reduced (for example, by fixing instrumentation), the limiter automatically recovers
and allows label values through again. No configuration changes are needed.

Recovery is not immediate. The limiter tracks cardinality over a sliding window (based on the registry's `stale_duration`). It takes at least that 
duration or longer for existing high-cardinality labels to age out before the label values are allowed through again.

### Estimate active series demand

When the active series limit is reached, the `tempo_metrics_generator_registry_active_series` metric no longer reflects the true demand. Use the `tempo_metrics_generator_registry_active_series_demand_estimate` metric to estimate what the active series count would be without the limit:

```promql
tempo_metrics_generator_registry_active_series_demand_estimate{}
```

This metric uses HyperLogLog estimation and has approximately 3% deviation from the actual cardinality. Use this to determine if you need to increase limits or reduce cardinality.

### Span name sanitization

If `span_name` is one of the highest-cardinality labels in your setup, the `span_name_sanitization` option can reduce it by grouping similar span names and replacing variable segments. For example, `GET /users/123` and `GET /users/456` are both mapped to `GET /users/<_>`.

To evaluate the potential impact without modifying metrics, set `span_name_sanitization` to `dry_run`:

```yaml
overrides:
  defaults:
    metrics_generator:
      span_name_sanitization: "dry_run"
```

After a few minutes, compare the demand estimate against current active series:

```promql
tempo_metrics_generator_registry_post_sanitization_demand_estimate{}
```

If this value is significantly lower than `tempo_metrics_generator_registry_active_series`, switch to `enabled` to apply the reduction.

After you enable the option, use the following metric to confirm spans are being sanitized:

```promql
rate(tempo_metrics_generator_registry_spans_sanitized_total{}[5m])
```

If this rate is zero after enabling, the DRAIN model hasn't found patterns yet. This is expected for workloads with already-consistent span naming. The model trains continuously and adapts as new span names arrive.

For more details on configuration and usage, refer to [Reduce cardinality with span name sanitization](/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/metrics-generator/reduce-cardinality/).

### Remote write failures

For any number of reasons, the generator may fail a write to the remote write target. Use the following metrics to
determine if that's happening:

```
sum(rate(prometheus_remote_storage_samples_failed_total{}[1m]))
sum(rate(prometheus_remote_storage_samples_dropped_total{}[1m]))
sum(rate(prometheus_remote_storage_exemplars_failed_total{}[1m]))
sum(rate(prometheus_remote_storage_exemplars_dropped_total{}[1m]))
```

## Service graph metrics

Service graphs have additional configuration which can impact the quality of the output metrics.

### Expired edges

The following metrics can be used to determine how many edges are failing to find a match.
The expired edge only includes those edges that are expired and have no matching information to generate a service graph edge.

Rate of edges that have expired without a match:

```
sum(rate(tempo_metrics_generator_processor_service_graphs_expired_edges{}[1m]))
```

The `unmatched_span_kind` label identifies the kind of span that did not find its matching span.
Use it to determine which side of an edge is missing:

```promql
sum by (unmatched_span_kind) (
  rate(tempo_metrics_generator_processor_service_graphs_expired_edges{}[1m])
)
```

For example, `unmatched_span_kind="SPAN_KIND_SERVER"` means that server spans are arriving without matching client spans.
This can indicate discarded client spans or incorrect instrumentation,
such as a server span directly parenting another server span.

Rate of all edges:

```
sum(rate(tempo_metrics_generator_processor_service_graphs_edges{}[1m]))
```

If you are seeing a large number of edges expire without a match, consider adjusting the `wait` setting. This
controls how long the metrics generator waits to find a match before it gives up.

```yaml
metrics_generator:
  processor:
    service_graphs:
      wait: 10s
```

### Service graph max items

The service graph processor has a maximum number of edges it tracks at once to limit the total amount of memory the processor uses.
To determine if edges are being dropped due to this limit, check:

```
sum(rate(tempo_metrics_generator_processor_service_graphs_dropped_spans{}[1m]))
```

Use `max_items` to adjust the maximum amount of edges tracked:

```yaml
metrics_generator:
  processor:
    service_graphs:
      max_items: 10000
```
