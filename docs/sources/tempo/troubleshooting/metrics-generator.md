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
- Reviewing the [service graph documentation](https://grafana.com/docs/tempo/<TEMPO_VERSION>/metrics-generator/service_graphs/) to understand how they are built.

If everything seems acceptable from these two perspectives, consider the following topics to help resolve general issues with all metrics and span metrics specifically.

## All metrics

This section covers metrics for all metrics related to the metrics-generator.

### Dropped spans in the distributor

The distributor has a queue of outgoing spans to the metrics-generators.
If the queue is full, then the distributor drops spans before they reach the generator. Use the following metric to determine if that's happening:

```
sum(rate(tempo_distributor_queue_pushes_failures_total{}[1m]))
```

### Failed pushes to the generator

For any number of reasons, the distributor can fail a push to the generators. Use the following metric to
determine if that's happening:

```
sum(rate(tempo_distributor_metrics_generator_pushes_failures_total{}[1m]))
```

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
