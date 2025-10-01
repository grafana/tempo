---
title: TraceQL metrics sampling guide
menuTitle: Sampling guide
description: Optimize TraceQL metrics query performance using sampling hints
weight: 500
keywords:
  - TraceQL metrics
  - sampling
  - performance optimization
  - query optimization
---

# TraceQL metrics sampling guide

{{< docs/shared source="tempo" lookup="traceql-metrics-admonition.md" version="<TEMPO_VERSION>" >}}

TraceQL metrics sampling enables faster query execution by processing a subset of trace data while maintaining acceptable accuracy.
Sampling delivers 2-4x performance improvements for heavy aggregation queries.

## Overview

The sampling addresses the challenge of balancing query performance with data accuracy when working with large-scale trace datasets.
Sampling intelligently selects a representative subset of data for processing, making it particularly valuable for:

- Real-time dashboards requiring fast refresh rates
- Exploratory data analysis where approximate results accelerate insights
- Resource-constrained environments with limited compute capacity
- Large-scale deployments processing terabytes of trace data daily

Refer to the [TraceQL metrics documentation](https://grafana.com/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/metrics-queries/) to learn more. 

## Before you begin

TraceQL metrics sampling requires:

- Tempo 2.8+ with TraceQL metrics enabled
- `local-blocks` processor configured in metrics-generator ([documentation](/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/metrics-queries/configure-traceql-metrics/))
- Grafana 10.4+ or Grafana Cloud for UI integration

## Choose a sampling method

Select a sampling method: 

- Adaptive sampling
- Fixed span sampling
- Fixed trace sampling
### Adaptive sampling: `with(sample=true)`

Adaptive sampling automatically determines the optimal sampling strategy based on query characteristics. It switches between span-level and trace-level sampling as needed and adjusts sampling rates dynamically.

```traceql
{ resource.service.name="checkout-service" } | rate() with(sample=true)
{ status=error } | count_over_time() by (resource.service.name) with(sample=true)
```

**Best for:** Most queries.  Specifically, all queries returning a single series, and cases where the dynamic sampling rate is important ,such as when the traffic has large variations across time or is not known in advance.

**Limitations:** May under-sample rare events depending on the query, if it returns time series with a large difference between the most common and rarest events.

### Fixed span sampling: `with(span_sample=0.xx)`

Fixed span sampling selects the specified percentage of spans.

```traceql
{ status=error } | rate() by (resource.service.name) with(span_sample=0.1)
```

**Best for:** Exact control over accuracy and speed when the data characteristics are known in advance.

**Limitations:** May miss important events during low-volume periods and not optimal for naturally selective queries.

### Fixed trace sampling: `with(trace_sample=0.xx)`

Fixed trace sampling selects complete traces for analysis, preserving trace context and relationships between spans within the same request flow.

```traceql
{ } >> { status=error }  | rate() by (resource.service.name) with(trace_sample=0.1)
```

**Best for:** Trace-level aggregations, service dependency mapping, and error correlation analysis.

**Limitations:** Not as accurate as span-level sampling when trace sizes vary significantly.  Only use for queries requiring it, such as structural or spanset correlation, and prefer adaptive or span-level sampling for all others.


