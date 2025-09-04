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

TraceQL metrics sampling is a performance optimization feature that enables faster query execution by processing a subset of trace data while maintaining acceptable accuracy. This guide provides essential information for implementing sampling in your TraceQL metrics queries.

## Introduction and overview

TraceQL metrics sampling addresses the challenge of balancing query performance with data accuracy when working with large-scale trace datasets. Traditional TraceQL metrics queries can be resource-intensive, especially when aggregating data across millions of spans or thousands of traces over extended time periods.

Sampling intelligently selects a representative subset of data for processing, delivering 2-4x performance improvements for heavy aggregation queries while maintaining 95%+ accuracy for most use cases. This makes sampling particularly valuable for:

- **Real-time dashboards** requiring fast refresh rates
- **Exploratory data analysis** where approximate results accelerate insights
- **Resource-constrained environments** with limited compute capacity
- **Large-scale deployments** processing terabytes of trace data daily

Grafana Tempo supports three complementary sampling approaches: adaptive sampling that automatically optimizes based on query characteristics, fixed span-level sampling for precise control over span selection, and trace-level sampling optimized for trace-wide aggregations.

## Before you begin

TraceQL metrics sampling requires:

- Tempo 2.8+ with TraceQL metrics enabled
- `local-blocks` processor configured in metrics-generator
- Grafana 10.4+ or Grafana Cloud for UI integration

## Sampling methods deep dive

### Adaptive sampling: `with(sample=true)`

Adaptive sampling represents the most sophisticated sampling approach, automatically determining the optimal sampling strategy based on query selectivity, data volume, and result distribution. The sampler analyzes query patterns and adjusts sampling rates dynamically to maintain accuracy while maximizing performance gains.

The adaptive sampler monitors query execution in real-time, tracking the rate at which matching spans are found. When data is abundant (high match rates), it reduces the sampling percentage to maintain performance. For rare events or selective queries, it maintains higher sampling rates or disables sampling entirely to preserve signal.

#### Syntax and examples

```traceql
// Basic adaptive sampling for service rate metrics
{ resource.service.name="checkout-service" } | rate() with(sample=true)

// Adaptive sampling with grouping
{ status=error } | count_over_time() by (resource.service.name) with(sample=true)

// Complex query with adaptive sampling
{ resource.service.name="api-gateway" && span.http.method="POST" }
| quantile_over_time(duration, .95, .99) by (span.http.route) with(sample=true)
```

#### Adaptive sampling use cases

- **Heavy aggregation queries** like `{ } | rate()` that process millions of spans
- **Dashboard queries** where consistent performance matters more than exact precision
- **User-driven exploration** when users need quick insights from large datasets
- **Multi-service analysis** with unpredictable data volumes across services

#### Adaptive sampling characteristics

- Automatically switches between span-level and trace-level sampling based on query type
- Uses probabilistic sampling with reservoir sampling for quantile calculations
- Applies back pressure when downstream processing becomes a bottleneck
- Scales sampling factors based on actual data encountered, not estimated volumes

#### Adaptive sampling limitations

- May over-sample rare events, reducing performance benefits for needle-in-haystack queries
- Results are deterministic within the same block but may vary across different blocks as new data arrives
- Less suitable for precise alerting scenarios requiring consistent thresholds

### Fixed span sampling: `with(span_sample=0.xx)`

Fixed span sampling provides deterministic control over sampling rates by selecting a specified percentage of spans regardless of query characteristics. This approach offers predictable performance improvements and consistent approximation levels across different queries.

The span sampler applies probabilistic sampling at the span level using consistent hashing of span IDs. This ensures reproducible sampling decisions—the same span is consistently included or excluded across different query executions. The sampling occurs early in the query pipeline, reducing I/O and processing overhead.

#### Fixed span sampling syntax

```traceql
// Sample 10% of spans for error rate analysis
{ status=error } | rate() by (resource.service.name) with(span_sample=0.1)

// Sample 5% of spans for latency percentiles
{ span.http.method="GET" } | quantile_over_time(duration, .95) with(span_sample=0.05)

// Sample 1% of spans for high-level service overview
{ } | count_over_time() by (resource.service.name) with(span_sample=0.01)
```

#### Fixed span sampling use cases

- **Consistent approximation** when you need predictable accuracy levels
- **Large-scale monitoring** where 1-5% sampling provides sufficient signal
- **Baseline metrics** for capacity planning and trend analysis
- **Cost optimization** in cloud environments where query costs scale with data processed

#### Fixed span sampling characteristics

- Deterministic sampling based on span ID hashing
- Linear performance scaling with sampling percentage
- Preserves temporal distribution of spans
- Compatible with all TraceQL metrics functions

#### Fixed span sampling limitations

- Fixed sampling may miss important events during low-volume periods
- Not optimal for queries with naturally low selectivity
- May under-represent short-lived or infrequent services

### Fixed trace sampling: `with(trace_sample=0.xx)`

Fixed trace sampling operates at the trace level, selecting complete traces for analysis rather than individual spans. This approach is particularly effective for queries that require full trace context or perform trace-wide aggregations.

#### Fixed trace sampling syntax

```traceql
// Sample 10% of traces for service dependency analysis
{ } | count() by (resource.service.name) with(trace_sample=0.1)

// Sample 5% of traces for end-to-end latency analysis
{ trace:duration > 1s } | avg_over_time(trace:duration) with(trace_sample=0.05)

// Sample traces for error correlation analysis
{ status=error } | count() with(trace_sample=0.2)
```

#### Fixed trace sampling use cases

- **Trace-level aggregations** using functions like `count()` that operate on whole traces
- **Service dependency mapping** where trace context is essential
- **Error correlation analysis** requiring complete request flows
- **Distributed system analysis** where span relationships matter

#### Fixed trace sampling characteristics

- Maintains complete trace context for selected traces
- Optimal for queries using trace-level attributes (trace:duration, trace:rootService)
- Preserves causal relationships between spans
- More efficient than span sampling for trace-wide analyses

#### Fixed trace sampling limitations

- May provide poor accuracy for span-level metrics if traces have variable span counts
- Less effective for queries focused on specific span types or services
- Can introduce bias if trace volumes vary significantly across services

## Implementation guide

### Get started with sampling

#### Step 1: Verify prerequisites

Check Tempo version (requires 2.8+):

```bash
curl -s http://your-tempo-endpoint:3200/api/status/buildinfo | jq '.version'
```

Verify local-blocks processor is enabled:

```bash
curl -s http://your-tempo-endpoint:3200/status/config | grep -A5 "local_blocks"
```

Test basic TraceQL metrics functionality:

```bash
curl -G "http://your-tempo-endpoint:3200/api/metrics/query" \
  --data-urlencode 'q={ } | rate()' \
  --data-urlencode 'since=5m'
```

#### Step 2: Baseline performance measurement

Before implementing sampling, establish baseline metrics for your most common queries.

Time a typical service overview query:

```bash
time curl -G "http://your-tempo-endpoint:3200/api/metrics/query_range" \
  --data-urlencode 'q={ } | rate() by (resource.service.name)' \
  --data-urlencode 'since=1h' \
  --data-urlencode 'step=1m'
```

Record: execution time, bytes processed, concurrent user capacity.

#### Step 3: Initial sampling implementation

Start with adaptive sampling on non-critical queries:

```traceql
// Convert existing queries gradually
// Before:
{ resource.service.name="api-gateway" } | rate() by (span.http.route)

// After:
{ resource.service.name="api-gateway" } | rate() by (span.http.route) with(sample=true)
```

### Query migration strategies

#### Systematic migration approach

1. **Development environment first**: Test all sampling configurations in non-production
2. **Dashboard queries**: Migrate dashboard panels before alerting queries
3. **Exploratory queries**: Apply sampling to user-driven analysis queries
4. **Monitoring queries**: Carefully evaluate accuracy requirements before migration

#### Migration checklist

- [ ] Document current query performance baselines
- [ ] Identify queries suitable for sampling (heavy aggregations)
- [ ] Test sampling accuracy against known results
- [ ] Update documentation with sampling rationale
- [ ] Configure monitoring for sampling effectiveness
- [ ] Plan rollback procedures for accuracy issues

### Grafana integration

**Dashboard panel configuration:**

```json
{
  "expr": "{ resource.service.name=\"frontend\" } | rate() by (span.http.method) with(sample=true)",
  "legendFormat": "{{span.http.method}}"
}
```

**Variable template integration:**

```traceql
// Use Grafana variables with sampling
{ resource.service.name="$service" } | rate() with(sample=true)

// Conditional sampling based on time range
{ resource.service.name="$service" } | rate()
  with(sample=$__range_s > 3600 ? true : false)
```

**Alert rule considerations:**

For alerting, prefer exact queries or high-accuracy sampling:

```traceql
// Critical alerts: avoid sampling
{ resource.service.name="payment-service" && status=error } | rate()

// Warning alerts: adaptive sampling acceptable
{ resource.service.name="payment-service" } | rate() with(sample=true) > 100
```

### Configuration optimization

**Frontend configuration for sampling:**

```yaml
query_frontend:
  metrics:
    # Optimize for sampled queries
    concurrent_jobs: 1500
    target_bytes_per_job: 1.5e+08 # 150MB
    max_exemplars: 1000

    # Increased concurrency since sampling reduces per-job processing
    max_outstanding_per_tenant: 2000
```

**Backend optimization:**

```yaml
querier:
  # Sampling reduces memory pressure
  max_concurrent_queries: 500

  # Faster timeouts possible with sampling
  search:
    query_timeout: 2m # Reduced from 5m
```

### Monitor sampling effectiveness

**Key metrics to track:**

- Query execution time distribution
- Accuracy correlation coefficients
- Resource utilization (CPU, memory, I/O)
- User experience metrics (dashboard load times)

**Sampling-specific monitoring:**

```yaml
# Add custom metrics for sampling effectiveness
- name: tempo_sampling_accuracy_correlation
  help: Correlation between sampled and exact results
  type: histogram
  buckets: [0.8, 0.85, 0.9, 0.95, 0.98, 1.0]

- name: tempo_sampling_performance_gain
  help: Performance improvement from sampling
  type: histogram
  buckets: [1.0, 1.5, 2.0, 3.0, 5.0, 10.0]
```

### A/B testing methodology

**Comparative analysis setup:**

```bash
#!/bin/bash

QUERY='{ resource.service.name="api" } | rate() by (span.http.method)'
START=$(date -d '1 hour ago' +%s)
END=$(date +%s)

echo "Testing exact query..."
time curl -G "http://tempo:3200/api/metrics/query_range" \
  --data-urlencode "q=${QUERY}" \
  --data-urlencode "start=${START}" \
  --data-urlencode "end=${END}" \
  --data-urlencode "step=1m" > exact_results.json

echo "Testing sampled query..."
time curl -G "http://tempo:3200/api/metrics/query_range" \
  --data-urlencode "q=${QUERY} with(sample=true)" \
  --data-urlencode "start=${START}" \
  --data-urlencode "end=${END}" \
  --data-urlencode "step=1m" > sampled_results.json

python3 compare_results.py exact_results.json sampled_results.json
```

## Best practices and recommendations

### Query design patterns for optimal sampling

#### Design principle 1: Favor broad queries over narrow ones

Sampling works best with queries that naturally match many spans, allowing statistical methods to maintain accuracy.

```traceql
// Preferred: Broad query with grouping
{ } | rate() by (resource.service.name, span.http.method) with(sample=true)

// Less optimal: Multiple narrow queries
{ resource.service.name="service-a" } | rate() with(sample=true)
{ resource.service.name="service-b" } | rate() with(sample=true)
```

#### Design principle 2: Align sampling method with aggregation scope

Match your sampling strategy to the scope of your metrics functions.

```traceql
// Span-level aggregations: use span sampling
{ span.http.method="POST" } | avg_over_time(duration) with(span_sample=0.1)

// Trace-level aggregations: use trace sampling
{ } | count() by (trace:rootService) with(trace_sample=0.1)

// Complex aggregations: use adaptive sampling
{ } | quantile_over_time(duration, .95, .99) by (resource.service.name) with(sample=true)
```

#### Design principle 3: Consider temporal patterns in sampling decisions

Account for traffic patterns and seasonal variations in your sampling strategy.

```yaml
# Grafana variable for time-aware sampling
sampling_rate: >
  $__range_s > 86400 ? 0.01 :    // >24h: 1% sampling
  $__range_s > 3600 ? 0.05 :     // >1h: 5% sampling
  0.1                            // <1h: 10% sampling
```

### Dashboard strategy and panel optimization

#### High-frequency panels (real-time monitoring)

- Use adaptive sampling for consistent performance
- Implement longer refresh intervals (30s+) to reduce query load
- Cache frequently accessed sampled results

```json
{
  "title": "Service Request Rate",
  "expr": "{ } | rate() by (resource.service.name) with(sample=true)",
  "refresh": "30s",
  "cacheTimeout": "1m"
}
```

#### Overview dashboards (service health)

- Apply consistent sampling across all panels
- Use trace sampling for service-level metrics
- Implement drill-down to exact queries for investigation

#### Historical analysis dashboards

- Use lower sampling rates (1-5%) for trend analysis
- Combine multiple time ranges with appropriate sampling
- Document sampling rates in panel descriptions

### Alerting considerations and guidelines

#### Critical alerts: Avoid sampling

For alerts that trigger operational responses, prioritize accuracy over performance.

```yaml
# Critical: Payment processing failures
- alert: PaymentServiceErrors
  expr: >
    { resource.service.name="payment" && status=error } | rate() > 0.05
  # No sampling - accuracy required for operational response

# Warning: General error rate trends
- alert: ServiceErrorTrend
  expr: >
    { status=error } | rate() by (resource.service.name) with(sample=true) > 0.1
  # Sampling acceptable for trend monitoring
```

#### Alert design patterns

- Use exact queries for critical business metrics
- Apply adaptive sampling for resource and performance alerts
- Implement multi-tier alerting (sampled warnings, exact critical alerts)
- Document sampling decisions in alert descriptions

### Data retention and historical analysis strategy

#### Sampling strategy by time horizon

- **Real-time (0-1h):** Adaptive sampling or 10%+ fixed rates
- **Recent history (1h-1d):** 5-10% sampling for most analyses
- **Historical trends (1d+):** 1-5% sampling sufficient for capacity planning
- **Long-term analysis (30d+):** 0.1-1% sampling for trend identification

#### Data lifecycle considerations

```yaml
# Adjust sampling based on data age
query_strategy:
  recent_data: # Last 24 hours
    default_sampling: "with(sample=true)"
    accuracy_requirement: ">95%"

  historical_data: # >24 hours old
    default_sampling: "with(span_sample=0.05)"
    accuracy_requirement: ">90%"

  archival_analysis: # >30 days old
    default_sampling: "with(span_sample=0.01)"
    accuracy_requirement: ">80%"
```

### Team workflows and operational integration

#### Development workflow integration

- Include sampling considerations in query review processes
- Establish sampling standards for different query types
- Create query templates with appropriate sampling configurations
- Document sampling rationale in monitoring-as-code repositories

#### Operational procedures

```markdown
## Query Performance Checklist

- [ ] Does this query process >1GB of data per execution?
- [ ] Is ~5% accuracy acceptable for this use case?
- [ ] Will this query run more than once per minute?
- [ ] Can we use adaptive sampling, or do we need fixed rates?

## Sampling Decision Tree

1. Critical alert or exact measurement needed? → No sampling
2. Dashboard or trend analysis? → Adaptive sampling
3. Historical analysis or capacity planning? → Fixed sampling (1-5%)
4. Cost optimization or initial exploration? → Low fixed sampling (0.1-1%)
```

#### Monitoring and maintenance

- Establish sampling effectiveness monitoring
- Schedule periodic accuracy validation against known baselines
- Plan sampling strategy reviews during capacity planning cycles
- Create runbooks for sampling-related incidents

#### Training and knowledge sharing

- Include sampling concepts in observability training programs
- Share sampling success stories and lessons learned
- Maintain internal documentation of sampling best practices
- Establish sampling expertise within teams

### Performance optimization guidelines

#### Resource allocation for sampling

- Increase query-frontend concurrency when implementing sampling
- Reduce per-job resource limits since sampling decreases processing requirements
- Monitor cache hit rates - sampling may change cache effectiveness
- Plan for adaptive sampling memory overhead during query initialization

#### Scaling considerations

- Sampling effectiveness improves with larger datasets
- Multi-tenant deployments benefit from per-tenant sampling strategies
- Geographic distribution may require region-specific sampling tuning
- Consider sampling in capacity planning for future growth

By following these practices, organizations can successfully integrate TraceQL metrics sampling into their observability workflows, achieving significant performance improvements while maintaining the data quality needed for effective monitoring and analysis.
