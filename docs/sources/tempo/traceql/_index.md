---
title: TraceQL
menuTitle: TraceQL
description: Learn about TraceQL, Tempo's query language for traces
weight: 450
aliases:
  - /docs/tempo/latest/traceql/
keywords:
  - Tempo query language
  - query language
  - TraceQL
---

# TraceQL

Inspired by PromQL and LogQL, TraceQL is a query language designed for selecting traces in Tempo. Currently, TraceQL query can select traces based on the following:

- Span and resource attributes, timing, and duration
- Basic aggregates: `count()` and `avg()`

Read the blog post, "[Get to know TraceQL](https://grafana.com/blog/2023/02/07/get-to-know-traceql-a-powerful-new-query-language-for-distributed-tracing/)," for an introduction to TraceQL and its capabilities.

{{< vimeo 796408188 >}}

For information on where the language is headed, see [future work](architecture).
The TraceQL language uses similar syntax and semantics as [PromQL](https://grafana.com/blog/2020/02/04/introduction-to-promql-the-prometheus-query-language/) and [LogQL](https://grafana.com/docs/loki/latest/logql/), where possible.

TraceQL requires Tempo’s Parquet columnar format to be enabled. For information on enabling Parquet, refer to the [Apache Parquet backend](https://grafana.com/docs/tempo/latest/configuration/parquet/) Tempo documentation.

## TraceQL query editor

With Tempo 2.0, you can use the TraceQL query editor in the Tempo data source to build queries and drill-down into result sets. The editor is available in Grafana’s Explore interface. For more information, refer to [TraceQL query editor]({{< relref "query-editor" >}}).

<p align="center"><img src="assets/query-editor-http-method.png" alt="Query editor showing request for http.method" /></p>

## Construct a TraceQL query

In TraceQL, a query is an expression that is evaluated on one trace at a time. The query is structured as a set of chained expressions (a pipeline). Each expression in the pipeline selects or discards spansets from being included in the results set. For example:

```
{ span.http.status_code >= 200 && span.http.status_code < 300 } | count() > 2
```

In this example, the search reduces traces to those spans where:

* `http.status_code` is in the range of `200` to `299` and
* the number of matching spans within a trace is greater than two.

Queries select sets of spans and filter them through a pipeline of aggregators and conditions. If, for a given trace, this pipeline produces a spanset then it is included in the results of the query.


## Selecting spans

TraceQL differentiates between two types of span data: intrinsics, which are fundamental to spans, and attributes, which are customizable key-value pairs. You can use intrinsics and attributes to build filters and select spans.

In TraceQL, curly brackets `{}` always select a set of spans from the current trace. They are commonly paired with a condition to reduce the spans being passed in.


### Intrinsic fields

Intrinsic fields are fundamental to spans. These fields can be referenced when selecting spans. Note that custom attributes are prefixed with `.`, `span.` or `resource.` whereas intrinsics are typed directly.

The following table shows the current intrinsic fields:

| **Field**     | **Type**    | **Definition**                                                  | **Example**            |
|---------------|-------------|-----------------------------------------------------------------|------------------------|
| status        | status enum | status: error, ok, or unset                                     | { status = ok }        |
| duration      | duration    | end - start time of the span                                    | { duration > 100ms }   |
| name          | string      | operation or span name                                          | { name = "HTTP POST" } |
| kind          | kind enum   | kind: server, client, producer, consumer, internal, unspecified | { kind = server }      |

### Attribute fields

There are two types of attributes: span attributes and resource attributes. By expanding a span in the Grafana UI, you can see both its span attributes (1 in the screenshot) and resource attributes (2 in the screenshot).

<p align="center"><img src="assets/span-resource-attributes.png" alt="Example of span and resource  attributes." /></p>

Attribute fields are derived from the span and can be customized. Process and span attribute types are [defined by the attribute itself](https://github.com/open-telemetry/opentelemetry-proto/blob/b43e9b18b76abf3ee040164b55b9c355217151f3/opentelemetry/proto/common/v1/common.proto#L30-L38), whereas intrinsic fields have a built-in type. You can refer to dynamic attributes (also known as tags) on the span or the span's resource.

Attributes in a query start with a span scope (for example, `span.http`) or resource scope (for example, `resource.namespace`)  depending on what you want to query. This provides significant performance benefits because it allows Tempo to only scan the data you are interested in.

To find traces with the `GET HTTP` method, your query could look like this:

```
{ span.http.method = "GET" }
```

For more information about attributes and resources, refer to the [OpenTelemetry Resource SDK](https://opentelemetry.io/docs/reference/specification/resource/sdk/).
#### Examples

Find traces that passed through the `prod` namespace:
```
{ resource.namespace = "prod" }
```

Find any database connection string that goes to a Postgres or MySQL database:
```
{ span.db.system =~ "postgresql|mysql" }
```

### Unscoped attribute fields

Attributes can be unscoped if you are unsure if the requested attribute exists on the span or resource. When possible, use scoped instead of unscoped attributes.

For example, to find traces with an attribute of `http.status` set to `200`:
```
{ .http.status = 200 }
```

To find traces where the `namespace` is set to `prod`:
```
{ .namespace = "prod" }
```


### Comparison operators

Comparison operators are used to test values within an expression.

The implemented comparison operators are:

- `=` (equality)
- `!=` (inequality)
- `>` (greater than)
- `>=` (greater than or equal to)
- `<` (less than)
- `<=` (less than or equal to)
- `=~` (regular expression)

For example, to find all traces where an `http.status` attribute in a span are greater than `400` but less than equal to `500`:

```
{ span.http.status >= 400 && span.http.status < 500 }
```

Find all traces where the `http.method` attribute is either `GET` or `DELETE`:

```
{ span.http.method =~ “DELETE|GET” }
```

### Field expressions

Fields can also be combined in various ways to allow more flexible search criteria. A field expression is a composite of multiple fields that define all of the criteria that must be matched to return results.

#### Examples

Find traces with "success" `http.status` codes:

```
{ span.http.status >= 200 && span.http.status < 300 }
```

Find traces where a `DELETE` HTTP method was used and the instrinsic span status was not OK:

```
{ span.http.method = "DELETE" && status != ok }
```

Both expressions require all conditions to be true on the same span. The entire expression inside of a pair of `{}` must be evaluated as true on a single span for it to be included in the result set.

In the above example, if a span includes an `.http.method` attribute set to `DELETE` where the span also includes a `status` attribute set to `ok`, the trace would not be included in the returned results.

## Combining spansets

Spanset operators let you combine two sets of spans using and (`&&`) as well as union (`||`).

- `{condA} && {condB}`
- `{condA} || {condB}`


For example, to find a trace that went through two specific regions:

```
{ resource.region = "eu-west-0" } && { resource.region = "eu-west-1" }
```

Note the difference between the previous example and this one:

```
{ resource.region = "eu-west-0" && resource.region = "eu-west-1" }
```

The second expression returns no traces because it's impossible for a single span to have a `resource.region` attribute that is set to both region values at the same time.

## Aggregators

So far, all of the example queries expressions have been about individual spans. You can use aggregate functions to ask questions about a set of spans. These currently consist of:

- `count` - The count of spans in the spanset.
- `avg` - The average of a given attribute or intrinsic for a spanset.

Aggregate functions allow you to carry out operations on matching results to further refine the traces returned. For more information on planned future work, refer to [How TraceQL works]({{< relref "architecture" >}}).

For example, to find traces where the total number of spans is greater than `10`:

```
count() > 10
```

Find traces where the average duration of the spans in a trace is greater than `20ms`:

```
avg(duration) > 20ms
```

For example, find traces that have more than 3 spans with an attribute `http.status` with a value of `200`:

```
{ span.http.status = 200 } | count() > 3
```

## Arithmetic

TraceQL supports arbitrary arithmetic in your queries. This can be useful to make queries more human readable:
```
{ span.bytesProcessed > 10 * 1024 * 1024 }
```
to compare the ratios of two span attributes:
```
{ span.bytesProcessed < span.jobsProcessed * 10 }
```
or anything else that comes to mind.

## Examples

Find any trace with a span attribute or resource attribute `namespace` set to `prod`:

```
{ .namespace = "prod" }
```

Find any trace with a resource attribute `namespace` set to `prod`:

```
{ resource.namespace = "prod" }
```

Find any trace with a `namespace` attribute set to `prod` and `http.status` attribute set to `200`:

```
{ .namespace = "prod" && .http.status = 200 }
```

Find any trace where spans within it have a `namespace` attribute set to `prod` and an `http.status` attribute set to `200`. In previous examples, all conditions had to be true on one span. These conditions can be true on either different spans or the same spans.

```
{ .namespace = "prod" } && { .http.status = 200 }
```

Find any trace where any span has an `http.method` attribute set to `GET` as well as a `status` attribute set to `ok`, where any other span also exists that has an `http.method` attribute set to `DELETE`, but does not have a `status` attribute set to `ok`:

```
{ .http.method = "GET" && status = ok } && { .http.method = "DELETE" && status != ok }
```
