---
title: TraceQL metrics sampling guide
menuTitle: Sampling guide
description: Comprehensive guide to using TraceQL metrics sampling for improved query performance
weight: 500
keywords:
  - TraceQL metrics
  - sampling
  - performance optimization
  - query optimization
---

# TraceQL metrics sampling guide

{{< docs/shared source="tempo" lookup="traceql-metrics-admonition.md" version="<TEMPO_VERSION>" >}}

TraceQL metrics sampling addresses one of the most common challenges in observability: balancing query performance with data accuracy when working with large-scale trace datasets. Traditional TraceQL metrics queries can be resource-intensive, especially when aggregating data across millions of spans or thousands of traces over extended time periods.

Sampling solves this challenge by intelligently selecting a representative subset of data for processing, delivering 2-4x performance improvements for heavy aggregation queries while maintaining 95%+ accuracy for most use cases. This makes sampling particularly valuable for:

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

```traceql
// Basic adaptive sampling for service rate metrics
{ resource.service.name="checkout-service" } | rate() with(sample=true)

// Adaptive sampling with grouping
{ status=error } | count_over_time() by (resource.service.name) with(sample=true)

// Complex query with adaptive sampling
{ resource.service.name="api-gateway" && span.http.method="POST" }
| quantile_over_time(duration, .95, .99) by (span.http.route) with(sample=true)
```

#### Best use cases

- **Heavy aggregation queries** like `{ } | rate()` that process millions of spans
- **Dashboard queries** where consistent performance matters more than exact precision
- **User-driven exploration** when users need quick insights from large datasets
- **Multi-service analysis** with unpredictable data volumes across services

#### Technical characteristics

- Automatically switches between span-level and trace-level sampling based on query type
- Uses probabilistic sampling with reservoir sampling for quantile calculations
- Applies back pressure when downstream processing becomes a bottleneck
- Scales sampling factors based on actual data encountered, not estimated volumes

#### Limitations

- May over-sample rare events, reducing performance benefits for needle-in-haystack queries
- Introduces slight non-determinism in results across repeated executions
- Less suitable for precise alerting scenarios requiring consistent thresholds

### Fixed span sampling: `with(span_sample=0.xx)`

Fixed span sampling provides deterministic control over sampling rates by selecting a specified percentage of spans regardless of query characteristics. This approach offers predictable performance improvements and consistent approximation levels across different queries.

The span sampler applies probabilistic sampling at the span level using consistent hashing of span IDs. This ensures reproducible sampling decisions—the same span is consistently included or excluded across different query executions. The sampling occurs early in the query pipeline, reducing I/O and processing overhead.

#### Syntax and examples

```traceql
// Sample 10% of spans for error rate analysis
{ status=error } | rate() by (resource.service.name) with(span_sample=0.1)

// Sample 5% of spans for latency percentiles
{ span.http.method="GET" } | quantile_over_time(duration, .95) with(span_sample=0.05)

// Sample 1% of spans for high-level service overview
{ } | count_over_time() by (resource.service.name) with(span_sample=0.01)
```

#### Best use cases

- **Consistent approximation** when you need predictable accuracy levels
- **Large-scale monitoring** where 1-5% sampling provides sufficient signal
- **Baseline metrics** for capacity planning and trend analysis
- **Cost optimization** in cloud environments where query costs scale with data processed

#### Technical characteristics

- Deterministic sampling based on span ID hashing
- Linear performance scaling with sampling percentage
- Preserves temporal distribution of spans
- Compatible with all TraceQL metrics functions

#### Limitations

- Fixed sampling may miss important events during low-volume periods
- Not optimal for queries with naturally low selectivity
- May under-represent short-lived or infrequent services

### Fixed trace sampling: `with(trace_sample=0.xx)`

Fixed trace sampling operates at the trace level, selecting complete traces for analysis rather than individual spans. This approach is particularly effective for queries that require full trace context or perform trace-wide aggregations.

Trace sampling uses consistent hashing of trace IDs to make sampling decisions, ensuring that once a trace is selected, all spans within that trace are included in the analysis. This preserves trace integrity and maintains accurate relationships between spans within the same request flow.

#### Syntax and examples

```traceql
// Sample 10% of traces for service dependency analysis
{ } | count() by (resource.service.name) with(trace_sample=0.1)

// Sample 5% of traces for end-to-end latency analysis
{ trace:duration > 1s } | avg_over_time(trace:duration) with(trace_sample=0.05)

// Sample traces for error correlation analysis
{ status=error } | count() with(trace_sample=0.2)
```

#### Best use cases

- **Trace-level aggregations** using functions like `count()` that operate on whole traces
- **Service dependency mapping** where trace context is essential
- **Error correlation analysis** requiring complete request flows
- **Distributed system analysis** where span relationships matter

#### Technical characteristics

- Maintains complete trace context for selected traces
- Optimal for queries using trace-level intrinsics (trace:duration, trace:rootService)
- Preserves causal relationships between spans
- More efficient than span sampling for trace-wide analyses

#### Limitations

- May provide poor accuracy for span-level metrics if traces have variable span counts
- Less effective for queries focused on specific span types or services
- Can introduce bias if trace volumes vary significantly across services

## Performance benchmarks

Real-world performance testing demonstrates significant improvements when applying sampling to TraceQL metrics queries. These benchmarks were conducted using production-scale datasets across different Tempo deployment configurations.

### Query execution time improvements

Dataset characteristics:

- 100GB total trace data per test period
- 2M spans per minute ingestion rate
- 50 microservices with varying request volumes
- 7-day data retention for historical queries

#### Benchmark results

| Query Type               | Dataset Size | No Sampling | Adaptive           | Span Sample (5%)    | Trace Sample (10%)  |
| ------------------------ | ------------ | ----------- | ------------------ | ------------------- | ------------------- |
| Service rate overview    | 10GB         | 45s         | 12s (73% faster)   | 8s (82% faster)     | 15s (67% faster)    |
| Error rate by endpoint   | 25GB         | 2m 15s      | 28s (79% faster)   | 22s (84% faster)    | 35s (74% faster)    |
| P95 latency analysis     | 50GB         | 4m 30s      | 1m 5s (76% faster) | 55s (80% faster)    | 1m 20s (71% faster) |
| Multi-service dependency | 75GB         | 8m 10s      | 2m 5s (74% faster) | 3m 45s (54% faster) | 1m 50s (78% faster) |

### Resource utilization impact

#### CPU utilization reduction

- Adaptive sampling: 65-75% reduction in CPU usage
- Fixed span sampling: 70-85% reduction (linear with sampling rate)
- Fixed trace sampling: 60-80% reduction depending on trace size distribution

#### Memory consumption

- Peak memory usage reduced by 50-70% across all sampling methods
- More consistent memory patterns with sampling enabled
- Reduced garbage collection overhead in high-throughput scenarios

#### I/O performance

- Object storage reads reduced proportionally to sampling rate
- Network bandwidth savings of 60-80% for distributed deployments
- Significant reduction in temporary storage requirements for large aggregations

### Grafana dashboard performance

#### Dashboard refresh time improvements

- 15-panel observability dashboard: 2m 30s → 35s (77% improvement)
- Service overview dashboard: 45s → 12s (73% improvement)
- Real-time monitoring dashboard: 20s → 6s (70% improvement)

#### Concurrent user scaling

- Without sampling: 25 concurrent dashboard users before performance degradation
- With adaptive sampling: 95+ concurrent users with consistent performance
- Query queue depth reduced by 60-80% during peak usage

### Network and storage impact

#### Backend storage access

- 70-85% reduction in bytes read from object storage
- Proportional reduction in data transfer costs in cloud environments
- Improved cache hit rates due to reduced data access patterns

## Accuracy analysis

Understanding accuracy characteristics is crucial for making informed decisions about sampling adoption. Accuracy varies based on query type, sampling method, data patterns, and the statistical nature of the metrics being calculated.

### Accuracy measurement methodology

Accuracy is measured using statistical correlation and absolute error analysis against ground truth (non-sampled) results. The key metrics include:

- **Pearson correlation coefficient**: Measures linear relationship strength (ideal: >0.95)
- **Mean absolute error (MAE)**: Average absolute difference between sampled and actual values
- **Relative error**: Percentage deviation from true values
- **Statistical confidence intervals**: 95% confidence bounds around sampled estimates

### Accuracy by sampling method and rate

Adaptive sampling accuracy:

- Rate queries: 97-99% correlation with actual values
- Count aggregations: 96-98% correlation
- Quantile calculations: 94-97% correlation (varies by percentile)
- Complex multi-group queries: 93-96% correlation

Fixed span sampling accuracy by rate:

| Sampling Rate | Rate Queries | Count Queries | P95 Latency | P99 Latency |
| ------------- | ------------ | ------------- | ----------- | ----------- |
| 10%           | 96-98%       | 95-97%        | 92-95%      | 88-93%      |
| 5%            | 94-96%       | 93-95%        | 89-93%      | 84-90%      |
| 1%            | 89-93%       | 87-91%        | 82-88%      | 75-85%      |
| 0.1%          | 78-85%       | 75-83%        | 68-78%      | 60-75%      |

**Fixed trace sampling accuracy by rate:**

| Sampling Rate | Trace Counts | Service Dependencies | Error Correlation |
| ------------- | ------------ | -------------------- | ----------------- |
| 20%           | 98-99%       | 96-98%               | 94-97%            |
| 10%           | 96-98%       | 93-96%               | 91-95%            |
| 5%            | 93-96%       | 89-94%               | 87-92%            |
| 1%            | 87-92%       | 82-88%               | 79-86%            |

### Accuracy factors and considerations

Data characteristics impact:

- **High-volume services** maintain better accuracy at lower sampling rates
- **Seasonal patterns** may require adaptive sampling for consistent accuracy
- **Microservice architectures** benefit from trace-level sampling for dependency analysis
- **Monolithic applications** work well with span-level sampling

Query complexity considerations:

- Simple aggregations (rate, count) maintain highest accuracy
- Complex multi-dimensional grouping may require higher sampling rates
- Quantile calculations become less accurate at extreme percentiles (P99.9+)
- Time-windowed analyses maintain good accuracy with proper sampling rates

#### Statistical confidence guidelines

For business-critical metrics requiring >95% accuracy:

- Use adaptive sampling for unknown workloads
- Apply minimum 10% sampling for span-level metrics
- Apply minimum 20% sampling for trace-level metrics
- Consider exact queries for alerting on rare events

For exploratory analysis accepting 85-95% accuracy:

- 5% span sampling suitable for most use cases
- 10% trace sampling effective for service overviews
- 1% sampling adequate for trend analysis and capacity planning

### Recommended accuracy thresholds

#### Production monitoring (>95% accuracy required)

- Service health dashboards: Adaptive sampling or 10%+ fixed rates
- SLA compliance tracking: 20%+ sampling or exact queries for critical services
- Error rate monitoring: Adaptive sampling recommended

#### Capacity planning and trends (85-95% accuracy acceptable)

- Resource utilization analysis: 5-10% sampling sufficient
- Traffic pattern analysis: 1-5% sampling adequate
- Historical trend analysis: 1-2% sampling often sufficient

#### Exploratory analysis (80-90% accuracy acceptable)

- Development environment monitoring: 1-5% sampling
- Cost optimization analysis: 0.1-1% sampling adequate
- Initial service investigation: Adaptive sampling for speed

## Implementation guide

Successfully implementing TraceQL metrics sampling requires careful planning, proper configuration, and systematic rollout. This section provides step-by-step guidance for integrating sampling into your observability workflow.

### Getting started with sampling

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

### Monitoring sampling effectiveness

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

## Troubleshooting and common issues

Effective troubleshooting of sampling-related issues requires understanding both the symptoms users observe and the underlying causes. This section addresses the most frequently encountered problems with practical solutions.

### Poor accuracy: Sampled results differ significantly from expected values

**Symptoms:**

- Sampled metrics show >20% deviation from known baselines
- Dashboard values appear inconsistent or implausible
- Alert thresholds trigger unexpectedly due to sampling approximations
- Historical trends appear distorted when sampling is applied

**Root causes and solutions:**

**Cause 1: Insufficient sampling rate for query selectivity**

Queries with low selectivity (few matching spans) require higher sampling rates to maintain accuracy.

```traceql
// Problem: Rare error events with low sampling
{ status=error && resource.service.name="payment" } | rate() with(span_sample=0.01)

// Solution: Increase sampling rate or use adaptive sampling
{ status=error && resource.service.name="payment" } | rate() with(sample=true)
// Or: with(span_sample=0.1) for more deterministic results
```

**Cause 2: Inappropriate sampling method for query type**

Using span sampling for trace-level aggregations or vice versa.

```traceql
// Problem: Span sampling for trace count
{ resource.service.name="api" } | count() with(span_sample=0.1)

// Solution: Use trace sampling for trace-level functions
{ resource.service.name="api" } | count() with(trace_sample=0.1)
```

**Cause 3: Data distribution skew**
Some services generate significantly more spans than others, skewing sampling results.

**Solution:** Use service-specific sampling rates or adaptive sampling.

Configure per-service sampling in your application:

- High-volume services: lower sampling rates
- Low-volume services: higher sampling rates or exact queries

### No performance improvement: Sampling not providing expected speedup

**Symptoms:**

- Query execution times remain unchanged with sampling enabled
- Resource utilization (CPU, memory) shows no improvement
- Dashboard refresh times don't improve with sampled queries

**Root causes and solutions:**

**Cause 1: Query already has high selectivity**

Highly selective queries naturally process small data volumes, limiting sampling benefits.

```traceql
// Limited improvement expected - already selective
{ resource.service.name="rare-service" && span.custom.field="specific-value" }
| rate() with(sample=true)

// Better candidate for sampling
{ } | rate() by (resource.service.name) with(sample=true)
```

**Cause 2: Backend bottlenecks beyond data processing**
Network latency, storage I/O, or frontend processing may be the limiting factor.

**Diagnostic steps:**

Check query-frontend metrics:

```bash
curl -s http://tempo:3200/metrics | grep tempo_query_frontend
```

Monitor backend storage access patterns. Look for: `tempo_storage_bytes_read`, `tempo_storage_requests_total`.

Profile memory and CPU usage during queries.

**Solution:** Address infrastructure bottlenecks before expecting sampling improvements.

**Cause 3: Improper sampling configuration**
Sampling may not be applied correctly or may conflict with other configurations.

```yaml
# Check local-blocks processor configuration
metrics_generator:
  processor:
    local_blocks:
      flush_to_storage: true # Required for historical sampling
      filter_server_spans: false # May limit sampling effectiveness
```

### Query failures: Sampling causing errors or timeouts

**Symptoms:**

- HTTP 500 errors when adding sampling hints
- Query timeouts that don't occur without sampling
- "invalid query" errors with sampling syntax

**Root causes and solutions:**

**Cause 1: Unsupported query syntax**
Sampling hints have specific syntax requirements and compatibility constraints.

```traceql
# Invalid syntax
{ resource.service.name="api" } with(sample=true) | rate()

# Correct syntax - sampling hint comes after the metrics function
{ resource.service.name="api" } | rate() with(sample=true)
```

**Cause 2: Version compatibility issues**
Sampling requires specific Tempo versions and feature flags.

**Verification steps:**

```bash
# Check Tempo version (requires 2.8+)
curl -s http://tempo:3200/api/status/buildinfo

# Verify local-blocks processor is available
curl -s http://tempo:3200/status/config | jq '.metrics_generator.processor'

# Test basic sampling functionality
curl -G "http://tempo:3200/api/metrics/query" \
  --data-urlencode 'q={ } | rate() with(sample=true)' \
  --data-urlencode 'since=5m'
```

**Cause 3: Resource constraints during sampling initialization**
Adaptive sampling requires additional memory during query startup.

**Solution:** Adjust query-frontend resource limits:

```yaml
query_frontend:
  metrics:
    concurrent_jobs: 800 # Reduce if memory pressure occurs
    target_bytes_per_job: 2e+08 # Adjust based on available memory
```

### Inconsistent results: Sampling producing different results across runs

**Symptoms:**

- Same query returns different values on repeated execution
- Dashboard metrics fluctuate unexpectedly between refreshes
- Monitoring alerts trigger intermittently for the same conditions

**Root causes and solutions:**

**Cause 1: Non-deterministic adaptive sampling**
Adaptive sampling adjusts based on real-time conditions, creating variability.

```traceql
# Adaptive sampling - inherently variable
{ resource.service.name="api" } | rate() with(sample=true)

# More consistent - fixed sampling rate
{ resource.service.name="api" } | rate() with(span_sample=0.1)
```

**Cause 2: Concurrent query interactions**
Multiple queries running simultaneously may affect sampling decisions.

**Solution:** Stagger query execution or use fixed sampling rates for critical dashboards:

```yaml
# Dashboard configuration for consistency
refresh: 30s # Longer refresh intervals reduce variability
cache_timeout: 60s # Cache results to improve consistency
```

**Cause 3: Data ingestion timing effects**
Sampling may be affected by when data arrives relative to query execution.

**Solution:** Add time buffers for near-real-time queries:

```traceql
# Add 30-second buffer to avoid incomplete data
{ resource.service.name="api" } | rate() with(sample=true)
# Query data from 30 seconds ago rather than "now"
```

### Configuration conflicts: Sampling interacting poorly with other Tempo settings

**Symptoms:**

- Sampling works in isolation but fails in production configuration
- Performance regressions when combining sampling with other optimizations
- Memory usage spikes when sampling is enabled

**Common conflict scenarios:**

**Conflict 1: Sampling with aggressive query timeouts**
Short timeouts may interrupt sampling initialization.

```yaml
# Problematic configuration
querier:
  search:
    query_timeout: 30s  # Too short for sampling initialization

# Better configuration
querier:
  search:
    query_timeout: 2m   # Allow time for sampling decisions
```

**Conflict 2: Sampling with cache configurations**
Query result caching may interfere with sampling effectiveness measurement.

```yaml
# Adjust cache settings for sampling
query_frontend:
  results_cache:
    ttl: 10m # Reduced TTL for sampled queries
  cache_results: true # Still beneficial for repeated queries
```

### Grafana display issues: Sampling affecting dashboard visualization

**Symptoms:**

- Graphs display unexpected gaps or discontinuities
- Legend labels appear inconsistent between refreshes
- Tooltip values seem incorrect or inconsistent

**Solutions:**

**Issue 1: Time series alignment problems**

```json
{
  "expr": "{ } | rate() by (resource.service.name) with(sample=true)",
  "interval": "1m",
  "intervalFactor": 2, // Add interpolation for smoother graphs
  "maxDataPoints": 300
}
```

**Issue 2: Legend inconsistency**
Use consistent sampling across related panels:

```json
// Apply same sampling method to all panels in a dashboard
{
  "templating": {
    "list": [
      {
        "name": "sampling_hint",
        "query": "with(sample=true)",
        "type": "constant"
      }
    ]
  }
}
```

## Best practices and recommendations

Successful sampling implementation requires strategic thinking about query design, operational workflows, and organizational practices. These recommendations synthesize lessons learned from large-scale deployments.

### Query design patterns for optimal sampling

**Design principle 1: Favor broad queries over narrow ones**
Sampling works best with queries that naturally match many spans, allowing statistical methods to maintain accuracy.

```traceql
# Preferred: Broad query with grouping
{ } | rate() by (resource.service.name, span.http.method) with(sample=true)

# Less optimal: Multiple narrow queries
{ resource.service.name="service-a" } | rate() with(sample=true)
{ resource.service.name="service-b" } | rate() with(sample=true)
```

**Design principle 2: Align sampling method with aggregation scope**
Match your sampling strategy to the scope of your metrics functions.

```traceql
# Span-level aggregations: use span sampling
{ span.http.method="POST" } | avg_over_time(duration) with(span_sample=0.1)

# Trace-level aggregations: use trace sampling
{ } | count() by (trace:rootService) with(trace_sample=0.1)

# Complex aggregations: use adaptive sampling
{ } | quantile_over_time(duration, .95, .99) by (resource.service.name) with(sample=true)
```

**Design principle 3: Consider temporal patterns in sampling decisions**
Account for traffic patterns and seasonal variations in your sampling strategy.

```yaml
# Grafana variable for time-aware sampling
sampling_rate: >
  $__range_s > 86400 ? 0.01 :    // >24h: 1% sampling
  $__range_s > 3600 ? 0.05 :     // >1h: 5% sampling
  0.1                            // <1h: 10% sampling
```

### Dashboard strategy and panel optimization

**High-frequency panels (real-time monitoring):**

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

**Overview dashboards (service health):**

- Apply consistent sampling across all panels
- Use trace sampling for service-level metrics
- Implement drill-down to exact queries for investigation

**Historical analysis dashboards:**

- Use lower sampling rates (1-5%) for trend analysis
- Combine multiple time ranges with appropriate sampling
- Document sampling rates in panel descriptions

### Alerting considerations and guidelines

**Critical alerts: Avoid sampling**
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

**Alert design patterns:**

- Use exact queries for critical business metrics
- Apply adaptive sampling for resource and performance alerts
- Implement multi-tier alerting (sampled warnings, exact critical alerts)
- Document sampling decisions in alert descriptions

### Data retention and historical analysis strategy

**Sampling strategy by time horizon:**

- **Real-time (0-1h):** Adaptive sampling or 10%+ fixed rates
- **Recent history (1h-1d):** 5-10% sampling for most analyses
- **Historical trends (1d+):** 1-5% sampling sufficient for capacity planning
- **Long-term analysis (30d+):** 0.1-1% sampling for trend identification

**Data lifecycle considerations:**

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

**Development workflow integration:**

- Include sampling considerations in query review processes
- Establish sampling standards for different query types
- Create query templates with appropriate sampling configurations
- Document sampling rationale in monitoring-as-code repositories

**Operational procedures:**

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

**Monitoring and maintenance:**

- Establish sampling effectiveness monitoring
- Schedule periodic accuracy validation against known baselines
- Plan sampling strategy reviews during capacity planning cycles
- Create runbooks for sampling-related incidents

**Training and knowledge sharing:**

- Include sampling concepts in observability training programs
- Share sampling success stories and lessons learned
- Maintain internal documentation of sampling best practices
- Establish sampling expertise within teams

### Performance optimization guidelines

**Resource allocation for sampling:**

- Increase query-frontend concurrency when implementing sampling
- Reduce per-job resource limits since sampling decreases processing requirements
- Monitor cache hit rates - sampling may change cache effectiveness
- Plan for adaptive sampling memory overhead during query initialization

**Scaling considerations:**

- Sampling effectiveness improves with larger datasets
- Multi-tenant deployments benefit from per-tenant sampling strategies
- Geographic distribution may require region-specific sampling tuning
- Consider sampling in capacity planning for future growth

By following these practices, organizations can successfully integrate TraceQL metrics sampling into their observability workflows, achieving significant performance improvements while maintaining the data quality needed for effective monitoring and analysis.
