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
This powerful feature allows for ad-hoc aggregation of any existing TraceQL query by any dimension available in your traces, much in the same way that LogQL metric queries create metrics from logs.

Traces are a unique observability signal that contain causal relationships between the components in your system.
Do you want to know how many database calls across all systems are downstream of your application?
What services beneath a given endpoint are currently failing?
What services beneath an endpoint are currently slow? TraceQL metrics can answer all these questions by parsing your traces in aggregate.

TraceQL metrics are powered by the TraceQL metrics API.

![Metrics visualization in Grafana](/media/docs/tempo/metrics-explore-sample-2.4.png)

## Enable and use TraceQL metrics

You can use the TraceQL metrics in Grafana with any existing or new Tempo data source.
This capability is available in Grafana Cloud and Grafana (10.4 and newer).

To enable TraceQL metrics, refer to [Configure TraceQL metrics](https://grafana.com/docs/tempo/latest/operations/traceql-metrics/) for more information.

## RED metrics, TraceQL, and PromQL

RED metrics represent:

- Rate, the number of requests per second
- Errors, the number of those requests that are failing
- Duration, the amount of time those requests take

For more information about RED method, refer to [The RED Method: how to instrument your services](/blog/2018/08/02/the-red-method-how-to-instrument-your-services/).

RED span metrics are created by the metrics-generator.
TraceQL metrics can expose the same RED metrics that would otherwise be queried using PromQL.

For more information on how to use TraceQL metrics to investigate issues, refer to [Solve problems with metrics queries](./solve-problems-metrics-queries).

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

TraceQL supports include `rate`, `count_over_time`, `quantile_over_time`, `histogram_over_time`, and `compare` functions.
These functions can be added as an operator at the end of any TraceQL query.

For detailed information, refer to [TraceQL metrics functions](./functions).