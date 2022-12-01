---
title: TraceQL
menuTitle: TraceQL
description: Learn about TraceQL, the query language for traces
weight: 450
aliases:
  - /docs/tempo/latest/traceql/
keywords:
  - Tempo query language
  - query language
  - TraceQL
---

# TraceQL

Inspired by PromQL and LogQL, TraceQL is a query language designed for selecting traces in Tempo. A TraceQL query can select traces based on:

- Span and resource attributes, timing, and duration
- Structural relationships between spans
- Aggregated data from the spans in a trace

The TraceQL language uses similar syntax and semantics as [PromQL](https://grafana.com/blog/2020/02/04/introduction-to-promql-the-prometheus-query-language/) and [LogQL](https://grafana.com/docs/loki/latest/logql/), where possible. TraceQL recognizes two types of data: intrinsics, which are fundamental to spans, and attributes, which are customizable key-value pairs.

TraceQL requires Tempo’s Parquet columnar format to be enabled. For information on enabling Parquet, refer to the [Apache Parquet backend](https://grafana.com/docs/tempo/latest/configuration/parquet/) Tempo documentation.

## How does it work?

The TraceQL engine connects the Tempo API handler with the storage layer. The TraceQL engine:

- Parses incoming requests and extract flattened conditions the storage layer can work with
- Pulls spansets from the storage layer and revalidates that the query matches each span
- Returns the search response

The default Tempo search reviews the whole trace. TraceQL provides a method for formulating precise queries so you can zoom in to the data you need. Query results are returned faster because the queries limit what is searched.

For an indepth look at TraceQL, read the [TraceQL: A first-of-its-kind query language to accelerate trace analysis in Tempo 2.0"](https://grafana.com/blog/2022/11/30/traceql-a-first-of-its-kind-query-language-to-accelerate-trace-analysis-in-tempo-2.0/) blog post by Trevor Jones.

For examples of query syntax, refer to [Perform a query]({{<relref "construct-query">}}).

{{< vimeo 773194063 >}}

## Active development and limitations

TraceQL is actively being developed. At this time, it is not production-ready.

TraceQL will be implemented in phases. The initial iteration of the TraceQL engine includes spanset selection and pipelines.

For more information about TraceQL’s design, refer to the [TraceQL Concepts design proposal](https://github.com/grafana/tempo/blob/main/docs/design-proposals/2022-04%20TraceQL%20Concepts.md).

### Known limitations

- Arithmetics are not implemented yet
- Scalar pipeline expressions do not allow static expressions or statics on the LHS. This seems odd but it's to remove conflicts with scalar filters. These are currently not allowed:
    - `(by(namespace) | count()) > 2 * 2`
    - `(by(namespace) | count()) * 2 > 2`
    - `2 < (by(namespace) | count())`
- Nested parents are currently not allowed

### Request access

Once TraceQL is available in Grafana Cloud as an experimental feature, you can open a ticket with Grafana Support to request access.