---
title: TraceQL metrics queries
menuTitle: TraceQL metrics queries
description: Learn about TraceQL metrics queries
weight: 600
keywords:
  - metrics query
  - TraceQL metrics
---

# TraceQL metrics queries

{{< docs/experimental product="TraceQL metrics" >}}

TraceQL metrics is an experimental feature in Grafana Tempo that creates metrics from traces.

Metric queries extend trace queries by applying a function to trace query results.
This powerful feature allows for adhoc aggregation of any existing TraceQL query by any dimension available in your traces, much in the same way that LogQL metric queries create metrics from logs.

Traces are a unique observability signal that contain causal relationships between the components in your system.
Do you want to know how many database calls across all systems are downstream of your application?
What services beneath a given endpoint are currently failing?
What services beneath an endpoint are currently slow? TraceQL metrics can answer all these questions by parsing your traces in aggregate.

![Metrics visualization in Grafana](/media/docs/tempo/metrics-explore-sample-2.4.png)

## Enable and use TraceQL metrics

You can use the TraceQL metrics in Grafana with any existing or new Tempo data source.
This capability is available in Grafana Cloud and Grafana (10.4 and newer).

To enable TraceQL metrics, refer to [Configure TraceQL metrics](https://grafana.com/docs/tempo/latest/operations/traceql-metrics/) for more information.

## Exemplars

Exemplars are a powerful feature of TraceQL metrics.
They allow you to see an exact trace that contributed to a given metric value.
This is particularly useful when you want to understand why a given metric is high or low.

Exemplars are available in TraceQL metrics for all functions.
To get exemplars, you need to configure it in the query-frontend with the parameter `query_frontend.metrics.exemplars`,
or pass a query hint in your query.

```
{ name = "GET /:endpoint" } | quantile_over_time(duration, .99) by (span.http.target) with (exemplars=true)
```

## Functions

TraceQL supports include `rate`, `count_over_time`, `quantile_over_time`, and `histogram_over_time` functions.
These functions can be added as an operator at the end of any TraceQL query.

`rate`
: Calculates the number of matching spans per second

`count_over_time`
: Counts the number of matching spans per time interval (see the `step` API parameter)

`min_over_time`
: Returns the minimum value of matching spans values per time interval (see the `step` API parameter)

`max_over_time`
: Returns the maximum value of matching spans values per time interval (see the `step` API parameter)

`avg_over_time`
: Returns the average value of matching spans values per time interval (see the `step` API parameter)

`quantile_over_time`
: The quantile of the values in the specified interval

`histogram_over_time`
: Evaluate frequency distribution over time. Example: `histogram_over_time(duration) by (span.foo)`

`compare`
: Used to split the stream of spans into two groups: a selection and a baseline. The function returns time-series for all attributes found on the spans to highlight the differences between the two groups.

### The `rate` function

The following query shows the rate of errors by service and span name.

```
{ status = error } | rate() by (resource.service.name, name)
```

This example calculates the rate of the erroring spans coming from the service `foo`.
Rate is a `spans/sec` quantity.

```
{ resource.service.name = "foo" && status = error } | rate()
```

Combined with the `by()` operator, this can be even more powerful.

```
{ resource.service.name = "foo" && status = error } | rate() by (span.http.route)
```

This example still rates the erroring spans in the service `foo` but the metrics have been broken
down by HTTP route.
This might let you determine that `/api/sad` had a higher rate of erroring
spans than `/api/happy`, for example.

### The `count_over_time`, `min_over_time`, `max_over_time` and `avg_over_time` functions

The `count_over_time()` let you counts the number of matching spans per time interval.

```
{ name = "GET /:endpoint" } | count_over_time() by (span.http.status_code)

```

The `min_over_time()` let you aggregate numerical values by computing the minimum value of them, such as the all important span duration.

```
{ name = "GET /:endpoint" } | min_over_time(duration) by (span.http.target)
```

Any numerical attribute on the span is fair game.

```
{ name = "GET /:endpoint" } | min_over_time(span.http.status_code)
```

The `max_over_time()` let you aggregate numerical values by computing the maximum value of them, such as the all important span duration.

```
{ name = "GET /:endpoint" } | max_over_time(duration) by (span.http.target)
```

```
{ name = "GET /:endpoint" } | max_over_time(span.http.status_code)
```

The `avg_over_time()` let you aggregate numerical values by computing the average value of them, such as the all important span duration.

```
{ name = "GET /:endpoint" } | avg_over_time(duration) by (span.http.target)
```

```
{ name = "GET /:endpoint" } | avg_over_time(event:cpu_seconds_tota)
```

### The `quantile_over_time` and `histogram_over_time` functions

The `quantile_over_time()` and `histogram_over_time()` functions let you aggregate numerical values, such as the all important span duration.
You can specify multiple quantiles in the same query.

```
{ name = "GET /:endpoint" } | quantile_over_time(duration, .99, .9, .5)
```

You can group by any span or resource attribute.

```
{ name = "GET /:endpoint" } | quantile_over_time(duration, .99) by (span.http.target)
```

Quantiles aren't limited to span duration.
Any numerical attribute on the span is fair game.
To demonstrate this flexibility, consider this nonsensical quantile on `span.http.status_code`:

```
{ name = "GET /:endpoint" } | quantile_over_time(span.http.status_code, .99, .9, .5)
```

### The `compare` function

This adds a new metrics function `compare` which is used to split the stream of spans into two groups: a selection and a baseline.
It returns time-series for all attributes found on the spans to highlight the differences between the two groups.
This is a powerful function that is best understood by looking at example outputs below:

The function is used like other metrics functions: when it's placed after any search query, and converts it into a metrics query:
`...any spanset pipeline... | compare({subset filters}, <topN>, <start timestamp>, <end timestamp>)`

Example:
```
{ resource.service.name="a" && span.http.path="/myapi" } | compare({status=error})
```
This function is generally run as an instant query.  It may return may exceed gRPC payloads when run as a query range.
#### Parameters

The `compare` function has four parameters:

1. Required. The first parameter is a spanset filter for choosing the subset of spans. This filter is executed against the incoming spans. If it matches, then the span is considered to be part of the selection. Otherwise, it is part of the baseline.  Common filters are expected to be things like `{status=error}` (what is different about errors?) or `{duration>1s}` (what is different about slow spans?)

2. Optional. The second parameter is the top `N` values to return per attribute. If an attribute exceeds this limit in either the selection group or baseline group, then only the top `N` values (based on frequency) are returned, and an error indicator for the attribute is included output (see below).  Defaults to `10`.

3. Optional. Start and End timestamps in Unix nanoseconds, which can be used to constrain the selection window by time, in addition to the filter. For example, the overall query could cover the past hour, and the selection window only a 5 minute time period in which there was an anomaly. These timestamps must both be given, or neither.

#### Output

The outputs are flat time-series for each attribute/value found in the spans.

Each series has a label `__meta_type` which denotes which group it is in, either `selection` or `baseline`.

Example output series:
```
{ __meta_type="baseline", resource.cluster="prod" } 123
{ __meta_type="baseline", resource.cluster="qa" } 124
{ __meta_type="selection", resource.cluster="prod" } 456   <--- significant difference detected
{ __meta_type="selection", resource.cluster="qa" } 125
{ __meta_type="selection", resource.cluster="dev"} 126  <--- cluster=dev was found in the highlighted spans but not in the baseline
```

When an attribute reaches the topN limit, there will also be present an error indicator.
This example means the attribute `resource.cluster` had too many values.
```
{ __meta_error="__too_many_values__", resource.cluster=<nil> }
```
