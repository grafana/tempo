# TraceQL Metrics Documentation

## Overview
TraceQL metrics functions convert trace queries into time-series metrics. These functions aggregate span data over time intervals to provide insights into system performance and behavior.

## Available Metrics Functions

### rate()
Calculates the number of matching spans per second.
```
{ <span_selection> } | rate()
```

Examples:
```
{ span:status = error } | rate()
{ resource.service.name = "frontend" } | rate() by (span.http.route)
```

### count_over_time()
Counts matching spans per time interval (controlled by `step` parameter).
```
{ <span_selection> } | count_over_time()
```

Examples:
```
{ span:name = "GET /api/users" } | count_over_time()
{ span.http.status_code >= 500 } | count_over_time() by (resource.service.name)
```

### sum_over_time()
Sums numeric values per time interval.
```
{ <span_selection> } | sum_over_time(<numeric_field>)
```

Examples:
```
{ } | sum_over_time(span.bytes_processed)
{ resource.service.name = "api" } | sum_over_time(span:duration)
```

### avg_over_time()
Calculates average values per time interval.
```
{ <span_selection> } | avg_over_time(<numeric_field>)
```

Examples:
```
{ } | avg_over_time(span:duration)
{ span.http.method = "POST" } | avg_over_time(span.http.response_size)
```

### min_over_time() and max_over_time()
Find minimum and maximum values per time interval.
```
{ <span_selection> } | min_over_time(<numeric_field>)
{ <span_selection> } | max_over_time(<numeric_field>)
```

Examples:
```
{ } | min_over_time(span:duration)
{ span.http.status_code >= 400 } | max_over_time(span:duration)
```

### quantile_over_time()
Calculate percentiles of numeric values.
```
{ <span_selection> } | quantile_over_time(<numeric_field>, <quantile1>, <quantile2>, ...)
```

Examples:
```
{ } | quantile_over_time(span:duration, .50, .95, .99)
{ resource.service.name = "api" } | quantile_over_time(span:duration, .99) by (span.http.route)
```

### histogram_over_time()
Evaluate frequency distribution over time.
```
{ <span_selection> } | histogram_over_time(<numeric_field>) by (<grouping_field>)
```

## Grouping with by()
Use `by()` to group metrics by specific attributes:
```
{ } | rate() by (resource.service.name)
{ span:status = error } | count_over_time() by (resource.service.name, span.http.route)
```

## Filtering Results with topk() and bottomk()
Limit results to top or bottom N series:
```
{ } | rate() by (resource.service.name) | topk(10)
{ } | rate() by (span.http.route) | bottomk(5)
```

## Comparison Operators on Results
Apply comparison operators to a metrics result to keep only the data points that meet a threshold. Supported operators are `>`, `>=`, `<`, `<=`, `=`, and `!=`. You can compare against integers, floats, and durations (for example `1s` or `500ms`).
```
{ } | rate() by (resource.service.name) > 10
{ span:name = "GET /:endpoint" } | avg_over_time(span:duration) > 1s
{ } | count_over_time() by (span:name) != 0
```

Data points that don't match are removed. If all data points in a series are removed, the entire series is dropped.

Comparison operators can be chained with `topk` and `bottomk` in any order, applied left to right:
```
{ } | rate() by (span.http.url) | topk(5) > 10
{ } | rate() by (span.http.url) > 0 | topk(5)
```

## Data Sampling
TraceQL metrics queries support sampling hints to optimize performance. Sampling hints only work with metrics queries (functions like `rate()`, `count_over_time()`, and so on).

- Dynamic sampling: `with(sample=true)` automatically determines the sampling strategy and amount based on the query.
- Fixed sampling: `with(sample=0.xx)`.
- Fixed span sampling: `with(span_sample=0.xx)` selects the given percentage of spans.
- Fixed trace sampling: `with(trace_sample=0.xx)` selects complete traces.

Examples:
```
{ resource.service.name = "frontend" } | rate() with(sample=true)
{ } | count_over_time() by (resource.service.name) with(sample=0.1)
{ span:status = error } | count_over_time() with(span_sample=0.1)
{ } | count_over_time() by (resource.service.name) with(trace_sample=0.05)
```

## Compare Function
Split spans into a selection group and a baseline group to highlight the differences between them. The function returns time-series for all attributes found on the spans.
```
{ <base_condition> } | compare({<selection_condition>}, <topN>, <start_timestamp>, <end_timestamp>)
```

Parameters:
- Required. A spanset filter that chooses the subset of spans. Matching spans are the selection; the rest are the baseline. Common filters are `{span:status = error}` (what is different about errors?) or `{span:duration > 1s}` (what is different about slow spans?).
- Optional. The top `N` values to return per attribute. Defaults to `10`.
- Optional. Start and end timestamps in Unix nanoseconds to constrain the selection window by time. Both must be given, or neither.

Example:
```
{ resource.service.name = "api" && span.http.path = "/users" } | compare({span:status = error})
```

The output is flat time-series for each attribute/value found in the spans. Each series carries a `__meta_type` label indicating the group. Possible values are `baseline`, `selection`, `baseline_total`, and `selection_total`:
```
{ __meta_type="baseline", resource.cluster="prod" } 123
{ __meta_type="selection", resource.cluster="prod" } 456
{ __meta_type="baseline_total", resource.cluster="prod" } 1000
{ __meta_type="selection_total", resource.cluster="prod" } 800
```

The `baseline_total` and `selection_total` series report the overall count across all values for each attribute, which helps calculate relative proportions.

When an attribute exceeds the top `N` limit, an error indicator series is included:
```
{ __meta_error="__too_many_values__", resource.cluster=<nil> }
```

`compare()` can't be combined with second-stage functions such as `topk`, `bottomk`, comparison operators, or arithmetic expressions. It's generally run as an instant query.

## Practical Examples

### Error Rate Monitoring
```
{ span:status = error } | rate() by (resource.service.name)
```

### Response Time Analysis
```
{ span.http.method = "GET" } | quantile_over_time(span:duration, .50, .95, .99) by (span.http.route)
```

### Service Performance Comparison
```
{ resource.service.name = "api" } | avg_over_time(span:duration) by (resource.deployment.environment)
```

### Top Slowest Endpoints
```
{ } | avg_over_time(span:duration) by (span.http.route) | topk(10)
```

### Error Distribution
```
{ span:status = error } | count_over_time() by (resource.service.name, span.http.status_code)
```

## Step Parameter
The `step` parameter controls the granularity of time-series data:
- Default: Automatically chosen based on query time range
- Custom: Use values like `30s`, `1m`, `5m`
- Configured via Grafana Explore or API

## Performance Tips
- Group by high-cardinality fields carefully
- Use topk/bottomk to limit result sets
- Consider using instant queries for single-point-in-time analysis