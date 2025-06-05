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

Inspired by PromQL and LogQL, TraceQL is a query language designed for selecting traces in Tempo. Currently, TraceQL query can select traces based on the following:

- Span and resource attributes, timing, and duration
- Basic aggregates: `count()`, `avg()`, `min()`, `max()`, and `sum()`

Read the blog post, [Get to know TraceQL](/blog/2023/02/07/get-to-know-traceql-a-powerful-new-query-language-for-distributed-tracing/), for an introduction to TraceQL and its capabilities.

{{< vimeo 796408188 >}}

The TraceQL language uses similar syntax and semantics as [PromQL](/blog/2020/02/04/introduction-to-promql-the-prometheus-query-language/) and [LogQL](/docs/loki/latest/logql/), where possible.

Check the [Tempo release notes](https://grafana.com/docs/tempo/<TEMPO_VERSION>/release-notes/) for the latest updates to TraceQL.

## Requirements

TraceQL requires the Parquet columnar format, which is enabled by default.
For information on Parquet, refer to the [Apache Parquet backend](http://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/parquet) documentation.

## Query using TraceQL

You can use TraceQL query editor and query builder in the Tempo data source to build queries and drill-down into result sets.
The editor and builder are available in the [Tempo data source](https://grafana.com/docs/grafana/<GRAFANA_VERSION>/datasources/tempo/) for Grafana Explore.

<p align="center"><img src="assets/query-editor-http-method.png" alt="Query editor showing request for http.method" /></p>

In addition, you can use Traces Drilldown to investigate your tracing data without writing TraceQL queries.
For more information, refer to the [Traces Drilldown](https://grafana.com/docs/grafana/<GRAFANA_VERSION>/explore/simplified-exploration/traces/) documentation.

### Stream query results

By streaming results to the client, you can start to look at traces matching your query before the entire query completes.

The GRPC streaming API endpoint in the query frontend allows a client to stream search results from Tempo.
The `tempo-cli` also uses this streaming endpoint.
For more information, refer to the [Tempo CLI documentation](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/tempo_cli/#query-api-command).

To use streaming in Grafana, you must have `stream_over_http_enabled: true` enabled in Tempo.
For information, refer to [Tempo GRPC API](https://grafana.com/docs/tempo/<TEMPO_VERSION>/api_docs/#tempo-grpc-api).

{{< docs/shared source="tempo" lookup="traceql-main.md" version="<TEMPO_VERSION>" >}}

## Experimental TraceQL metrics

TraceQL metrics are experimental, but easy to get started with.
Refer to [the TraceQL metrics](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/traceql-metrics/) documentation for more information.

You can also use TraceQL metrics queries.
For details, refer to [TraceQL metrics queries](https://grafana.com/docs/tempo/<TEMPO_VERSION>/traceql/metrics-queries/).