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

## Functions

TraceQL supports include `rate`, `count_over_time`, `quantile_over_time`, and `histogram_over_time` functions.
These functions can be added as an operator at the end of any TraceQL query.

`rate`
  : Provides the percentage of

## The `rate` function

The following query shows the rate of errors by service and span name.

```
{ status = error } | rate() by (resource.service.name, name)
```

This example calculates the rate of the erroring spans coming from the service `foo`. Rate is a `spans/sec` quantity.

```
{ resource.service.name = "foo" && status = error } | rate()
```

Combined with the `by()` operator, this can be even more powerful.

```
{ resource.service.name = "foo" && status = error } | rate() by (span.http.route)
```

This example still rates the erroring spans in the service `foo` but the metrics have been broken
down by HTTP route. This might let you determine that `/api/sad` had a higher rate of erroring
spans than `/api/happy`, for example.

### The `quantile_over_time` and `histogram_over_time` functions

The `quantile_over_time()` and `histogram_over_time()` functions let you aggregate numerical values, such as the all important span duration. Notice that you can specify multiple quantiles in the same query.

```
{ name = "GET /:endpoint" } | quantile_over_time(duration, .99, .9, .5)`
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