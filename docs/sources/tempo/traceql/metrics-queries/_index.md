---
title: TraceQL metrics queries
menuTitle: TraceQL metrics queries
description: Learn about TraceQL metrics queries
weight: 600
keywords:
  - metrics query
  - TraceQL metrics
refs:
  solve-problems-mq:
    - pattern: /docs/tempo/
      destination: https://grafana.com/docs/tempo/<TEMPO_VERSION>/traceql/metrics-queries/solve-problems-metrics-queries/
    - pattern: /docs/enterprise-traces/
      destination: https://grafana.com/docs/enterprise-traces/<ENTERPRISE_TRACES_VERSION>/traceql/metrics-queries/solve-problems-metrics-queries/
  mq-functions:
    - pattern: /docs/tempo/
      destination: https://grafana.com/docs/tempo/<TEMPO_VERSION>/traceql/metrics-queries/functions/
    - pattern: /docs/enterprise-traces/
      destination: https://grafana.com/docs/enterprise-traces/<ENTERPRISE_TRACES_VERSION>/traceql/metrics-queries/functions/
---

# TraceQL metrics queries

{{< docs/experimental product="TraceQL metrics" >}}

TraceQL metrics is an experimental feature in Grafana Tempo that creates metrics from traces.

Metric queries extend trace queries by applying a function to trace query results.
This powerful feature allows for ad hoc aggregation of any existing TraceQL query by any dimension available in your traces, much in the same way that LogQL metric queries create metrics from logs.

Traces are a unique observability signal that contain causal relationships between the components in your system.

TraceQL metrics can help answer questions like this:

* How many database calls across all systems are downstream of your application?
* What services beneath a given endpoint are currently failing?
* What services beneath an endpoint are currently slow?

TraceQL metrics can help you answer these questions by parsing your traces in aggregate.

TraceQL metrics are powered by the [TraceQL metrics API](https://grafana.com/docs/tempo/<TEMPO_VERSION>/api_docs/#traceql-metrics).

![Metrics visualization in Grafana](/media/docs/tempo/metrics-explore-sample-2.4.png)

## RED metrics, TraceQL, and PromQL

RED is an acronym for three types of metrics:

- Rate, the number of requests per second
- Errors, the number of those requests that are failing
- Duration, the amount of time those requests take

For more information about the RED method, refer to [The RED Method: how to instrument your services](/blog/2018/08/02/the-red-method-how-to-instrument-your-services/).

You can write TraceQL metrics queries to compute rate, errors, and durations over different groups of spans.

For more information on how to use TraceQL metrics to investigate issues, refer to [Solve problems with metrics queries](ref:solve-problems-mq).

## Enable and use TraceQL metrics

To use TraceQL metrics, you need to enable them on your Tempo database.
Refer to [Configure TraceQL metrics](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/traceql-metrics/) for more information.

From there, you can either query the TraceQL metrics API directly (for example, with `curl`) or using Grafana
(recommended).
To run TraceQL metrics queries in Grafana, you need Grafana Cloud or Grafana 10.4 or later.
No extra configuration is needed.
Use a Tempo data source that points to a Tempo database with TraceQL metrics enabled.

Refer to [Solve problems using metrics queries](ref:solve-problems-mq) for some real-world examples.

### Functions

TraceQL metrics queries currently include the following functions for aggregating over groups of spans: `rate`, `count_over_time`, `max_over_time`, `min_over_time`, `avg_over_time`, `quantile_over_time`, `histogram_over_time`, and `compare`.
These functions can be added as an operator at the end of any TraceQL query.

For detailed information and example queries for each function, refer to [TraceQL metrics functions](ref:mq-functions).

### Exemplars

Exemplars are a powerful feature of TraceQL metrics.
They allow you to see an exact trace that contributed to a given metric value.
This is particularly useful when you want to understand why a given metric is high or low.

Exemplars are available in TraceQL metrics for all range queries.
To get exemplars, you need to configure it in the query-frontend with the parameter `query_frontend.metrics.max_exemplars`,
or pass a query hint in your query.

Example:

```
{ span:name = "GET /:endpoint" } | quantile_over_time(duration, .99) by (span.http.target) with (exemplars=true)
```

{{< admonition type="note" >}}
TraceQL metric queries with exemplars aren't fully supported in Grafana Explore.
They will be supported in a future Grafana release.
{{< /admonition >}}
