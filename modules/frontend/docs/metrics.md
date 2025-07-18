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
{ name = "GET /api/users" } | count_over_time()
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

## Compare Function
Split spans into selection and baseline groups to highlight differences:
```
{ <base_condition> } | compare({<selection_condition>}, <topN>, <start_timestamp>, <end_timestamp>)
```

Example:
```
{ resource.service.name = "api" && span.http.path = "/users" } | compare({span:status = error})
```

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