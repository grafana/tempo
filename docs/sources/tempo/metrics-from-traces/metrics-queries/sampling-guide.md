---
title: TraceQL metrics sampling
menuTitle: TraceQL metrics sampling
description: Optimize TraceQL metrics query performance using sampling hints
weight: 500
keywords:
  - TraceQL metrics
  - sampling
  - performance optimization
  - query optimization
---

# TraceQL metrics sampling

{{< docs/shared source="tempo" lookup="traceql-metrics-admonition.md" version="<TEMPO_VERSION>" >}}

TraceQL metrics sampling dynamically and automatically chooses how to sample your tracing data to give you the highest quality signal with examining as little data as possible.
The overall performance improvement depends on the query. Heavy queries, such as `{ } | rate()`, show improvements of 2-4 times.

Sampling intelligently selects a representative subset of data for processing, making it particularly valuable for:

- Real-time dashboards requiring fast refresh rates
- Exploratory data analysis where approximate results accelerate insights
- Resource-constrained environments with limited compute capacity
- Large-scale deployments processing terabytes of trace data daily

Adaptive sampling was featured in the September 2025 Tempo community call. Watch the [recording](https://www.youtube.com/watch?v=7H8JX5FUw08) starting at the 12:00 minute mark to learn more.

Refer to the [TraceQL metrics documentation](https://grafana.com/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/metrics-queries/) to learn more.

{{< youtube id="fdmLmJMlUjI" start="720" >}}

## Sampling methods

There are three sampling methods available:

- Dynamic sampling using `with(sample=true)`, which automatically determines the optimal sampling strategy based on query characteristics.
- Fixed span sampling using `with(span_sample=0.xx)`, which selects the specified percentage of spans.
- Fixed trace sampling using `with(trace_sample=0.xx)`, which selects complete traces for analysis.
- Fixed probabilistic sampling using `with(sample=0.xx)`.

### How dynamic sampling works

Dynamic sampling, `with(sample=true)`, applies probabilistic sampling at the storage layer.
This sampling method uses an adaptive probabilistic approach that responds to how common spans and traces matching the query are.
This approach applies probabilistic sampling at the storage layer, for example, only inspecting `xx%` spans, or `xx%` traces, depending on the needs of the query.

When there is a lot of data, it lowers the sampling rate. When matches are rare it keeps the sampling rate higher, possibly never going below 100%. Therefore, the performance gain depends on the query.

This behavior can be overridden to focus more on fixed span sampling using `with(span_sample=0.xx)` or fixed trace sampling using `with(trace_sample=0.xx)`.

## Before you begin

TraceQL metrics sampling requires:

- Tempo 2.8+ with TraceQL metrics enabled
- `local-blocks` processor configured in metrics-generator ([documentation](/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/metrics-queries/configure-traceql-metrics/))
- Grafana 10.4+ or Grafana Cloud for UI integration

You can use the TraceQL query editor in the Tempo data source in Grafana or Grafana Cloud to run the sample queries.
Refer to [TraceQL queries in Grafana](https://grafana.com/docs/tempo/<TEMPO_VERSION>/traceql/query-editor/) for more information.

## Dynamic sampling using `with(sample=true)`

Dynamic sampling automatically determines the optimal sampling strategy based on query characteristics. It switches between span-level and trace-level sampling as needed and adjusts sampling rates dynamically.
The goal is for `with(sample=true)` to be safe to include in virtually any query, regardless of scale or selectivity.

```traceql
{ resource.service.name="checkout-service" } | rate() with(sample=true)
{ status=error } | count_over_time() by (resource.service.name) with(sample=true)
```

**Best for:** Most queries. Specifically, all queries returning a single series, and cases where the dynamic sampling rate is important, such as when the traffic has large variations across time or is not known in advance.

**Limitations:** May under-sample rare events depending on the query, if it returns time series with a large difference between the most common and rarest events.

## Fixed span sampling using `with(span_sample=0.xx)`

Fixed span sampling selects the specified percentage of spans.

```traceql
{ status=error } | rate() by (resource.service.name) with(span_sample=0.1)
```

**Best for:** Exact control over accuracy and speed when the data characteristics are known in advance.

**Limitations:** May miss important events during low-volume periods and not optimal for naturally selective queries.

## Fixed trace sampling using `with(trace_sample=0.xx)`

Fixed trace sampling selects complete traces for analysis, preserving trace context and relationships between spans within the same request flow.

```traceql
{ } >> { status=error }  | rate() by (resource.service.name) with(trace_sample=0.1)
```

**Best for:** Trace-level aggregations, service dependency mapping, and error correlation analysis.

**Limitations:** Not as accurate as span-level sampling when trace sizes vary significantly. Only use for queries requiring it, such as structural or spanset correlation, and prefer adaptive or span-level sampling for all others.
