---
Authors: Martin Disibio (@mdisibio)
Created: 2023 November
Last updated: 2023-11-13
---

# TraceQL Metrics

## Summary

This design document describes TraceQL language additions for extracting time-series metrics like error rates and latencies from traces.   

This document is **not** meant to be a complete language specification. Rather it is meant to invite the community to comment on the direction and capabilities.

### Table of Contents

- [Intro](#intro)
- [First Stage](#first-stage)
  - [Interval](#interval)
  - [Grouping](#grouping)
  - [Filtering](#filtering)
- [Additional Stages](#additional-stages)
- [Arithmetic](#arithmetic)
- [Notes](#notes)
  - [Step Interval](#step-interval)

## Intro

Let's look at an example straight away.  This query plots the error rate by service over time:  
`{status = error } | rate() by (resource.service.name)`

A normal TraceQL query finds matching spans and returns data points like time, duration, service name, status code, etc.  A metrics query at the core does the same thing: _find matching spans_, but instead of seeing individual results, they are aggregated over time to create time-series.  A metrics query is built by extending a regular query with metrics pipeline stages.  The structure looks like this:

`<span query...> | <metrics query...>`

An important understanding is that _any_ valid TraceQL query can be turned into a metrics query. This finds the call rates to service B but only when initiated upstream by service A:  
` {resource.service.name="A"} >> {resouce.service.name="B"} | rate()`


## First Stage

The first stage in a metrics query is responsible for turning spans into time series. This is a unique intersection between the two data types and has specialized aggregation functions.  We use the `..._over_time()` suffix to distinguish them from their counterparts.

| function                                   | description                |
---------------------------------------------| ---------------------------- 
|`rate()`                                    | The rate of spans per second
|`count_over_time()`                         | The count of spans
|`avg_over_time(<field>)`                    | The average of a numeric value like `duration` or [http.request.body.size](https://opentelemetry.io/docs/specs/semconv/attributes-registry/http/)
|`max_over_time(<field>)`                    | The maximum of a numeric value.
|`min_over_time(<field>)`                    | The minimum of a numeric value.
|`quantile_over_time(<field>, q1, q2, ...)`  | The quantile of a numeric value (e.g. p95). Multiple quantiles can be requested and a time-series is created for each. |

### Interval
All aggregations take an optional interval like `30s`, `5m`, `1h`. If not specified then the interval will automatically match the step interval of the query. The interval is the last parameter of each function.

`rate(5m)`  
`quantile_over_time(duration, 0.95, 5m)`

### Grouping
All aggregations can group by one or more attributes and generate a time-series for each combination of values.  

Plot the request rate of `/myapi` by user ID and HTTP status code:  
`{ span.http.path = "/myapi" } | rate() by (span.user_id, span.http.status_code)`

Plot the p95 duration of requests to service by HTTP endpoint:  
`{ resource.service.name = "myservice" } | quantile_over_time(duration, 0.95) by (span.http.path)`

### Filtering
Aggregations can be combined with comparison operations to filter the output.

Find the request rate per service but only if more than 1000 req/s:  
`{ } | rate() by (resource.service.name) > 1000`


## Additional Stages
Additional metrics stages can be added to further refine and transform the time-series. These stages receive time-series and generate new time-series so they are separate from the first stage.  

Most functions support the same ability to regroup using `by()` and comparison operators to refilter.  These can be thought of as instant functions that work across inputs at each point in time. Therefore they do not have an interval and do not alter the frequency of data points. If the original aggregation creates a data point every 60s, these will also output a data point every 60s.

| function                        | description                |
----------------------------------| ---------------------------- 
| `... \| max() [by(...)]`        | Maximum value for each point in time
| `... \| min() [by(...)]`        | Minimum value for each point in time
| `... \| avg() [by(...)]`        | Average across values for each point in time
| `... \| stddev() [by(...)]`     | Standard deviation across values for each point in time
| `... \| quantile(q) [by(...)]`  | Quantile across values for each point in time
| `... \| topk(N)`                | Return the top series for each point in time (grouping TBD)

Find the highest per-pod failure rate in each cluster:  
`{ status = error } | rate() by (cluster, pod) | max() by (cluster)`


## Arithmetic
Arithmetic operations between time-series can be used to calculate things like error rates.

Find the 5xx error rate for an endpoint:
```
({ span.http.path = "/myapi" && span.http.status_code >= 500 } | rate()) 
   /
({ span.http.path = "/myapi" | rate())
```

Operations:
* `*`
* `/`
* `+`
* `-`

## Notes

### Step Interval
The step interval is the explicit resolution of a query.  It is identical to the [Prometheus step interval](https://prometheus.io/docs/prometheus/latest/querying/api/#range-queries).  A step interval of `1m` means that a data point will be returned once every 60 seconds.  It is separate from the `interval` of an aggregation like `rate(1h)`.  The query `rate(1h)` with `step=1m` will still return a data point every 60 seconds but each data point is the rate smoothed over the previous hour.