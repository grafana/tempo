---
title: Construct a TraceQL query
menuTitle: Construct a query
description: Learn how to construct a TraceQL query
aliases:
  - /docs/tempo/latest/traceql/construct-query
weight: 100
keywords:
  - tempo query language
  - query language
  - TraceQL
---

# Construct a TraceQL query

In TraceQL, a query is an expression that is evaluated on one trace at a time. The query is structured as a set of chained expressions (a pipeline). Each expression in the pipeline selects or discards spansets from being included in the results set. For example:

```
{ .http.method = "POST" } | by(.http.target) | count() > 2
```

In this case, the search looks for any trace which includes a span attribute `http.method` set to `POST`, which is filtered by the attribute `http.target`, where two or more spans match that criteria.

Queries select sets of spans and filter them through a pipeline of aggregators and conditions. If a spanset is produced after evaluation on a trace, then this spanset (and by extension the trace) is included in the result set of the query.

## TraceQL query editor

With Tempo 2.0, you can use the TraceQL query editor in the Tempo data source to build queries and drill-down into result sets. The editor is available in Grafana’s Explore interface.

This screenshot shows the query and results from the search above, `{ .http.method = "POST" } | by(.http.target) | count() > 2`.

<p align="center"><img src="../assets/query-editor-http-method.png" alt="Query editor showing request for http.method" /></p>

Using the query editor, you can use the editor’s autocomplete suggestions to write queries. The editor detects span sets to provide relevant autocomplete options. It uses regular expressions (regex) to detect where it is inside a spanset and provide attribute names, scopes, intrinsic names, logic operators, or attribute values from Tempo's API, depending on what is expected for the current situation.

<p align="center"><img src="../assets/query-editor-auto-complete.png" alt="Query editor showing the auto-complete feature" /></p>

Query results are returned in a table. Selecting the Trace ID or Span ID provides more detailed information.

<p align="center"><img src="../assets/query-editor-results-span.png" alt="Query editor showing span results" /></p>

Selecting the trace ID from the returned results will open a trace diagram. Selecting a span from the returned results opens a trace diagram and reveals the relevant span in the trace diagram (above, the highlighted blue line).

## Selecting spans

In TraceQL, curly brackets `{}` always select a set of spans from the current trace. They are commonly paired with a condition to reduce the spans being passed in.

This simple query will be evaluated on every span of every trace, one at a time.

```
{ .http.status = 200 }
```

If the trace being evaluated contains no spans with an attribute `http.status` with the value `200`, then no spans will be selected and this trace will not appear in the result set.

If the trace does contain spans with an attribute `http.status` with the value `200`, then only those spans will be returned. The trace is reduced to only the set of spans that match the condition inside the `{}`. The result set will contain only this subset of spans matching the condition.

### Intrinsic fields

Intrinsic fields are fundamental to spans. Thes fields can be referenced when selecting spans.


| **Operation** | **Type** | **Definition**                        | **Example**            |
|---------------|----------|---------------------------------------|------------------------|
| status        | string   | status values are error, ok, or unset | { status = ok }        |
| duration      | duration | end - start time of the span          | { duration > 100ms }   |
| name          | string   | operation or span name                | { name = "HTTP POST" } |

### Attribute fields

Attribute fields are derived from the span and can be customized. Process and span attribute types are [defined by the attribute itself](https://github.com/open-telemetry/opentelemetry-proto/blob/b43e9b18b76abf3ee040164b55b9c355217151f3/opentelemetry/proto/common/v1/common.proto#L30-L38), whereas intrinsic fields have a built-in type. You can refer to dynamic attributes (also known as tags) on the span or the span's resource.

Attributes in a query start with a period and must end with a space. To find traces with the `GET HTTP` method, your query could look like this:

```
{ .http.method = "GET" }
```

Or like this:

```
{.http.method ="GET"}
```

#### Examples

Find traces that passed through the prod namespace:

```
{ .namespace = "prod" }
```

Find any database connection string that goes to a Postgres or MySQL database:

```
{ .db.system =~ "postgresql|mysql" }
```

### Scoped attribute fields

Attributes can be specifically scoped to either "span" or "resource". Specifying "span" or "resource" can result in significant performance benefits.

For example, to find traces with a span attribute of `http.status` set to `200`:
{ span.http.status = 200 }
To find traces where a the resource `namespace` is set to `prod`:
{ resource.namespace = "prod" }

### Scoped attribute fields

Attributes can be specifically scoped to either "span" or "resource". Specifying "span" or "resource" can result in significant performance benefits.

For example, to find traces with a span attribute of `http.status` set to `200`:

```
{ span.http.status = 200 }
```

To find traces where a the resource `namespace` is set to `prod`:

```
{ resource.namespace = "prod" }
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

For example, to find all traces where an `http.status` attribute in a span are greater than `300` but less than equal to `500`:

```
{ .http.status > 300 && .http.status < 500 }
```

Find all traces where the `http.method` attribute is either `GET` or `DELETE`:

```
{ .http.method =~ “DELETE|GET” }
```

### Field expressions

Fields can also be combined in various ways to allow more flexible search criteria. A field expression is a composite of multiple fields that define all of the criteria that must be matched to return results.

#### Examples

Find traces with "success" http status codes:

```
{ .http.status >= 200 && .http.status < 300 }
```

Find traces where a `DELETE` HTTP method was used and the status was not OK:

```
{ .http.method = "DELETE" && status != ok }
```

Both expressions require all conditions to be true on the same span. The entire expression inside of a pair of `{}` must be evaluated as true on a single span for it to be included in the result set.

In the above example, if a span includes an `.http.method` attribute set to `DELETE` where the span also includes a `status` attribute set to `ok`, the trace would not be included in the returned results.

## Combining spansets

Spanset operators let you combine two sets of spans using and (`&&`) as well as union (`||`).

- `{condA} && {condB}`
- `{condA} || {condB}`


For example, if you want to find a trace that went through two specific regions:

```
{ .region = "eu-west-0" } && { .region = "eu-west-1" }
```

Note the difference between the above and the following:

```
{ .region = "eu-west-0" && .region = "eu-west-1" }
```

The second expression returns no traces because it's impossible for a single span to have a `.region` attribute that is set to both region values at the same time.

## Aggregators

So far, all of the example queries expressions have been about individual spans. You can use aggregate functions to ask questions about a set of spans. These currently consist of:

- `count` - The total count across a given intrinsic or items.
- `avg` - The average across a given intrinsic or items.

Aggregate functions allow you to carry out operations on matching results to further refine the traces returned.

For example, to find traces where the total number of spans is greater than `10`:

```
count() > 10
```

Find traces where the average duration of the spans in a trace is greater than `20ms`:

```
avg(duration) > 20ms
```

## Expression pipelining

Pipelining lets you "pipe" a set of spans from one expression to the next. This is particularly useful if you want to perform an aggregate over a subset of a trace.

For example, find traces that have more than 3 spans with an attribute `http.status` with a value of `200`:

```
{ .http.status = 200 } | count() > 3
```

## Grouping

Grouping lets you take a trace and break it down into sets of spans that are evaluated by future pipeline entries. Each set of spans created by the group is individually evaluated by downstream expressions.

For example, find traces that have more than 5 spans in any region:

```
by(.region) | count() > 5
```

## Examples

Find any trace with a `namespace` attribute set to `prod`:

```
{ .namespace = "prod" }
```

Find any trace with a `namespace` attribute set to `prod` where the `http.status` is set to `200`:

```
{ .namespace = "prod" && .http.status = 200 }
```

Find any trace where any independent spans within it have a `namespace` attribute set to `prod` and an `http.status` attribute set to `200`:

```
{ .namespace = "prod" } && { .http.status = 200 }
```

Find any trace where any span has an `http.method` attribute set to `GET` as well as a `status` attribute set to `ok`, where any other span also exists that has an `http.method` attribute set to `DELETE`, but does not have a `status` attribute set to `ok`:

```
{ .http.method = "GET" && status = ok } && { .http.method = "DELETE" && status != ok }
```

Find any trace where the average client-span duration in a trace exceeds a threshold of one second:

```
{ span.kind = "client" } | avg(duration) > 1s
```

Find any trace that has a certain number of a given attribute in any namespace:

```
by(.namespace) | { .http.status = 500 } | count() > 5
```

Find any trace where any single span has the `http.method` attribute set to `GET` and the `endpoint` attribute is not set to `/login`, where the number of unique `endpoint` attributes is greater than 2 across all of the relevant spans:

```
{ .http.method = "GET" && .endpoint != "/login" } | by(.endpoint) | count() > 2
```
