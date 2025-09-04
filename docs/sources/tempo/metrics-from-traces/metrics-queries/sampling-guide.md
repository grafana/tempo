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

TraceQL metrics sampling is a performance optimization feature that enables faster query execution by processing a subset of trace data while maintaining acceptable accuracy. Sampling delivers 2-4x performance improvements for heavy aggregation queries.

## Overview

TraceQL metrics sampling addresses the challenge of balancing query performance with data accuracy when working with large-scale trace datasets. Sampling intelligently selects a representative subset of data for processing, making it particularly valuable for:

- Real-time dashboards requiring fast refresh rates
- Exploratory data analysis where approximate results accelerate insights
- Resource-constrained environments with limited compute capacity
- Large-scale deployments processing terabytes of trace data daily

## Prerequisites

TraceQL metrics sampling requires:

- Tempo 2.8+ with TraceQL metrics enabled
- `local-blocks` processor configured in metrics-generator
- Grafana 10.4+ or Grafana Cloud for UI integration

## Choose a sampling method

### Adaptive sampling: `with(sample=true)`

Adaptive sampling automatically determines the optimal sampling strategy based on query characteristics. It switches between span-level and trace-level sampling as needed and adjusts sampling rates dynamically.

```traceql
{ resource.service.name="checkout-service" } | rate() with(sample=true)
{ status=error } | count_over_time() by (resource.service.name) with(sample=true)
```

**Best for:** Heavy aggregation queries, dashboard queries, and multi-service analysis with unpredictable data volumes.

**Limitations:** May over-sample rare events and results vary across blocks as new data arrives.

### Fixed span sampling: `with(span_sample=0.xx)`

Fixed span sampling selects a specified percentage of spans using consistent hashing of span IDs. Provides predictable performance improvements and deterministic results.

```traceql
{ status=error } | rate() by (resource.service.name) with(span_sample=0.1)
```

**Best for:** Consistent approximation, large-scale monitoring, and cost optimization scenarios.

**Limitations:** May miss important events during low-volume periods and not optimal for naturally selective queries.

### Fixed trace sampling: `with(trace_sample=0.xx)`

Fixed trace sampling selects complete traces for analysis, preserving trace context and relationships between spans within the same request flow.

```traceql
{ } | count() by (resource.service.name) with(trace_sample=0.1)
```

**Best for:** Trace-level aggregations, service dependency mapping, and error correlation analysis.

**Limitations:** May provide poor accuracy for span-level metrics and can introduce bias if trace volumes vary significantly across services.

## Implement sampling

### Get started

1. **Verify prerequisites:** Check Tempo version and ensure local-blocks processor is enabled
2. **Start with adaptive sampling:** Apply `with(sample=true)` to non-critical queries first
3. **Measure performance:** Compare execution times before and after sampling
4. **Validate accuracy:** Test sampled results against exact results for critical queries

### Grafana integration

Use sampling in dashboard panels:

```json
{
  "expr": "{ resource.service.name=\"frontend\" } | rate() with(sample=true)"
}
```

For alerts, avoid sampling for critical alerts that trigger operational responses. Adaptive sampling is acceptable for warning alerts and trend monitoring.

### Configuration optimization

Increase query concurrency since sampling reduces per-job processing:

```yaml
query_frontend:
  metrics:
    concurrent_jobs: 1500
    target_bytes_per_job: 1.5e+08
```

## Best practices

### Query design

- **Use broad queries:** Sampling works best with queries that match many spans
- **Align sampling with aggregation scope:** Use span sampling for span-level aggregations, trace sampling for trace-level aggregations
- **Consider temporal patterns:** Adjust sampling rates based on data age and query frequency

### Select sampling rates by use case

- **Real-time monitoring (0-1h):** Adaptive sampling or 10%+ fixed rates
- **Recent analysis (1h-1d):** 5-10% sampling
- **Historical trends (1d+):** 1-5% sampling
- **Long-term analysis (30d+):** 0.1-1% sampling

### Decision framework

1. **Critical measurement needed?** → No sampling
2. **Dashboard or trend analysis?** → Adaptive sampling
3. **Historical analysis or capacity planning?** → Fixed sampling (1-5%)
4. **Cost optimization or exploration?** → Low fixed sampling (0.1-1%)

### Migration approach

1. Test all sampling configurations in development first
2. Migrate dashboard queries before alerting queries
3. Document sampling rationale and accuracy requirements
4. Configure monitoring for sampling effectiveness
5. Plan rollback procedures for accuracy issues

By following these practices, you can successfully integrate TraceQL metrics sampling into your observability workflows, achieving significant performance improvements while maintaining data quality for effective monitoring and analysis.
