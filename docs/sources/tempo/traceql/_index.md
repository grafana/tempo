---
title: TraceQL
menuTitle: TraceQL
description: Learn about TraceQL, Tempo's query language for traces
weight: 600
aliases:
  - /docs/tempo/latest/traceql/
keywords:
  - Tempo query language
  - query language
  - TraceQL
---

<!-- TraceQL pages are mounted in GET. Refer to params.yaml in the website repo. -->

# TraceQL

TraceQL is a query language designed for selecting traces in Tempo.

Distributed traces contain a wealth of information, and tools like auto-instrumentation make it easy to start capturing data. Extracting value from traces can be much harder.
For example, Tempo metrics-generator can aggregate traces into service graphs and span metrics, and exemplars allow you to navigate from a spike in API latency to a trace that contributed to that spike.
But traces can do so much more.

Traces are the flow of events throughout your components.
They have a tree structure—-with a root, branches, and leaves—-arbitrary key/value data at any location, and of course timestamps.
What new questions can be answered with this structure? More than just finding isolated events, can we find sequences of events?

For example, you can use traces to perform root cause analyses (RCA) on a service outage and use TraceQL to pinpoint the root cause. Refer to [Diagnose errors with traces](/docs/tempo/<TEMPO_VERSION>/solutions-with-traces/traces-diagnose-errors/#diagnose-errors-with-traces) for a use case example.

## Get started with TraceQL

Use these references to get started with TraceQL:

- Determine the information you want to query by understanding the [relationship of queries to trace structure](https://grafana.com/docs/tempo/<TEMPO_VERSION>/traceql/structure/) and spans.
- [Construct a query to locate the information](https://grafana.com/docs/tempo/<TEMPO_VERSION>/traceql/construct-traceql-queries/). Use the examples and language reference to learn about the syntax and semantics of TraceQL.

TraceQL uses similar syntax and semantics as [PromQL](/blog/2020/02/04/introduction-to-promql-the-prometheus-query-language/) and [LogQL](/docs/loki/latest/logql/), where possible.

{{< vimeo 796408188 >}}

## How can you use TraceQL?

You can use TraceQL queries using the command line or in Grafana with the query editor and query builder.
The query editor and builder are available in the [Tempo data source](https://grafana.com/docs/grafana/<GRAFANA_VERSION>/datasources/tempo/) for Grafana Explore.

In addition, you can use Traces Drilldown to investigate your tracing data without writing TraceQL queries.
For more information, refer to the [Traces Drilldown](https://grafana.com/docs/grafana/<GRAFANA_VERSION>/explore/simplified-exploration/traces/) documentation.

For more information, refer to [Write TraceQL queries in Grafana](http://grafana.com/docs/tempo/<TEMPO_VERSION>/traceql/query-editor/).

![Metrics visualization in Grafana](/media/docs/tempo/metrics-explore-sapmle-v2.7.png)

### Metrics from traces with TraceQL

TraceQL metrics generate metrics from traces and let you use TraceQL to query metrics.
Refer to [TraceQL metrics](https://grafana.com/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/metrics-queries/) for more information.

{{< docs/shared source="tempo" lookup="traceql-metrics-admonition.md" version="<TEMPO_VERSION>" >}}

## Resources

TraceQL requires the Parquet columnar format, which is the default block format for Tempo. Refer to the [Apache Parquet backend](http://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/parquet) documentation.

Refer to the [Tempo release notes](https://grafana.com/docs/tempo/<TEMPO_VERSION>/release-notes/) for the latest updates to TraceQL.
