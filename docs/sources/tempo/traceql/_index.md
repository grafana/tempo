---
title: Query with TraceQL
menuTitle: Query with TraceQL
description: Learn about TraceQL, Tempo's query language for traces
weight: 600
aliases:
  - /docs/tempo/latest/traceql/
keywords:
  - Tempo query language
  - query language
  - TraceQL
---

# Query with TraceQL

Inspired by PromQL and LogQL, TraceQL is a query language designed for selecting traces in Tempo. Currently, TraceQL query can select traces based on the following:

- Span and resource attributes, timing, and duration
- Basic aggregates: `count()`, `avg()`, `min()`, `max()`, and `sum()`

Read the blog post, [Get to know TraceQL](/blog/2023/02/07/get-to-know-traceql-a-powerful-new-query-language-for-distributed-tracing/), for an introduction to TraceQL and its capabilities.

{{< vimeo 796408188 >}}

For information on where the language is headed, see [future work]({{< relref "./architecture#future-work" >}}).
The TraceQL language uses similar syntax and semantics as [PromQL](/blog/2020/02/04/introduction-to-promql-the-prometheus-query-language/) and [LogQL](/docs/loki/latest/logql/), where possible.

Check the [release notes]({{< relref "../release-notes" >}}) for the latest updates to TraceQL.

## Requirements

TraceQL requires Tempo’s Parquet columnar format, which is enabled by default. For information on Parquet, refer to the [Apache Parquet backend]({{< relref "..//configuration/parquet" >}}) Tempo documentation.

## TraceQL queries in Grafana Explore

You can use the TraceQL query editor and query builder in the Tempo data source to build queries and drill-down into result sets.
The editor and builder are available in Grafana’s Explore interface.
For more information, refer to [Write TraceQL queries in Grafana]({{< relref "./query-editor" >}}).

<p align="center"><img src="assets/query-editor-http-method.png" alt="Query editor showing request for http.method" /></p>

### Stream query results

By streaming results to the client, you can start to look at traces matching your query before the entire query completes.

The [GRPC streaming API endpoint]({{< relref "../api_docs#tempo-grpc-api" >}}) in Tempo’s query frontend allows a client to stream search results from Tempo.
The `tempo-cli` also uses this streaming endpoint.
For more information, refer to the [Tempo CLI documentation]({{< relref "../operations/tempo_cli#query-api-command" >}}).

To use streaming in Grafana, you must have `stream_over_http_enabled: true` enabled in Tempo.
For information, refer to [Tempo GRPC API](https://grafana.com/docs/tempo/latest/api_docs/#tempo-grpc-api).

## Construct a TraceQL query

In TraceQL, a query is an expression that is evaluated on one trace at a time. The query is structured as a set of chained expressions (a pipeline). Each expression in the pipeline selects or discards spansets from being included in the results set. For example:

```
{ span.http.status_code >= 200 && span.http.status_code < 300 } | count() > 2
```

In this example, the search reduces traces to those spans where:

* `http.status_code` is in the range of `200` to `299` and
* the number of matching spans within a trace is greater than two.

Queries select sets of spans and filter them through a pipeline of aggregators and conditions.
If, for a given trace, this pipeline produces a spanset then it's included in the results of the query.

Refer to [TraceQL metrics queries](https://grafana.com/docs/tempo/<TEMPO_VERSION>/traceql/metrics-queries/) for examples of TraceQL metrics queries.

### Find traces of a specific operation

Let's say that you want to find traces of a specific operation, then both the operation name (the span attribute `name`) and the name of the service that holds this operation (the resource attribute `service.name`) should be specified for proper filtering.
In the example below, traces are filtered on the `resource.service.name` value `frontend` and the span `name` value `POST /api/order`:

```
{resource.service.name = "frontend" && name = "POST /api/orders"}
```

When using the same Grafana stack for multiple environments (for example, `production` and `staging`) or having services that share the same name but are differentiated though their namespace, the query looks like:

```
{
  resource.service.namespace = "ecommerce" &&
  resource.service.name = "frontend" &&
  resource.deployment.environment = "production" &&
  name = "POST /api/orders"
}
```

### Find traces having a particular outcome

This example finds all traces on the operation `POST /api/orders` that have a span that has errored:

```
{
  resource.service.name="frontend" &&
  name = "POST /api/orders" &&
  status = error
}
```

This example finds all traces on the operation `POST /api/orders` that return with an HTTP 5xx error:

```
{
  resource.service.name="frontend" &&
  name = "POST /api/orders" &&
  span.http.status_code >= 500
}
```

### Find traces that have a particular behavior

You can use query filtering on multiple spans of the traces.
This example locates all the traces of the `GET /api/products/{id}` operation that access a database. It's a convenient request to identify abnormal access ratios to the database caused by caching problems.

```
{span.service.name="frontend" && name = "GET /api/products/{id}"} && {.db.system="postgresql"}
```

### Find traces going through `production` and `staging` instances

This example finds traces that go through `production` and `staging` instances.
It's a convenient request to identify misconfigurations and leaks across production and non-production environments.

```
{ resource.deployment.environment = "production" } && { resource.deployment.environment = "staging" }
```

### Use structural operators

Find traces that include the `frontend` service, where either that service or a downstream service includes a span where an error is set.

```
{ resource.service.name="frontend" } >> { status = error }
```

Find all leaf spans that end in the `productcatalogservice`.

```
{ } !< { resource.service.name = "productcatalogservice" }
```

Find if `productcatalogservice` and `frontend` are siblings.

```
{ resource.service.name = "productcatalogservice" } ~ { resource.service.name="frontend" }
```

### Other examples

Find the services where the http status is 200, and list the service name the span belongs to along with returned traces.

```
{ span.http.status_code = 200 } | select(resource.service.name)
```

Find any trace with an unscoped `deployment.environment` attribute set to `production` and `http.status_code` attribute set to `200`:

```
{ .deployment.environment = "production" && span.http.status_code = 200 }
```

Find any trace where spans within it have a `deployment.environment` resource attribute set to `production` and a span `http.status_code` attribute set to `200`. In previous examples, all conditions had to be true on one span. These conditions can be true on either different spans or the same spans.

```
{ resource.deployment.environment = "production" } && { span.http.status_code = 200 }
```

Find any trace where any span has an `http.method` attribute set to `GET` as well as a `status` attribute set to `ok`, and where any other span has an `http.method` attribute set to `DELETE`, but doesn't have a `status` attribute set to `ok`:

```
{ span.http.method = "GET" && status = ok } && { span.http.method = "DELETE" && status != ok }
```

Find any trace with a `deployment.environment` attribute that matches the regex `prod-.*` and `http.status_code` attribute set to `200`:

```
{ resource.deployment.environment =~ "prod-.*" && span.http.status_code = 200 }
```

## Selecting spans

In TraceQL, curly brackets `{}` always select a set of spans from available traces.
Curly brackets are commonly paired with a condition to reduce the spans fetched.

TraceQL differentiates between two types of span data: intrinsics, which are fundamental to spans, and attributes, which are customizable key-value pairs.
You can use intrinsics and attributes to build filters and select spans.

{{< youtube id="aIDkPJ_e3W4" >}}

Intrinsic fields are fundamental to scopes.
Intrinsics are inherently present, as opposed to other key-value pairs (attributes) that are added by a developer.

Intrinsics are always indicated using a `<scope>:`.
Refer to the Intrinsics table for all current intrinsics.

Intrinsics example:
```
{ span:name = "foo" }
{ event:name = "foo" }
{ trace:id = "1234" }
{ link:traceID = "1234" }
```

Custom attributes are prefixed with `<scope>.`, such as `span.`,  `resource.` , `link.`, or `event`.
Resource has no intrinsic values.
It only has custom attributes.

Attributes are separated by a period (`.`), and intrinsic fields use a colon (`:`).
The `trace` scope is only an intrinsic and doesn't have any custom attributes at the trace level.

Attributes example:
```
{ span.foo = "bar" }
{ resource.foo = "bar" }
{ link.foo = "bar" }
{ event.foo = "bar" }
```

### Intrinsic fields

The following table shows the current available scoped intrinsic fields:

| **Field**                | **Type**    | **Definition**                                                  | **Example**                             |
| ------------------------ | ----------- | --------------------------------------------------------------- | --------------------------------------- |
| `span:status`            | status enum | status: error, ok, or unset                                     | `{ span:status = ok }`                  |
| `span:statusMessage`     | string      | optional text accompanying the span status                      | `{ span:statusMessage = "Forbidden" }`  |
| `span:duration`          | duration    | end - start time of the span                                    | `{ span:duration > 100ms }`             |
| `span:name`              | string      | operation or span name                                          | `{ span:name = "HTTP POST" }`           |
| `span:kind`              | kind enum   | kind: server, client, producer, consumer, internal, unspecified | `{ span:kind = server }`                |
| `span:id`                | string      | span id using hex string                                        | `{ span:id = "0000000000000001" }`      |
| `trace:duration`         | duration    | max(end) - min(start) time of the spans in the trace            | `{ trace:duration > 100ms }`            |
| `trace:rootName`         | string      | if it exists, the name of the root span in the trace            | `{ trace:rootName = "HTTP GET" }`       |
| `trace:rootService`      | string      | if it exists, the service name of the root span in the trace    | `{ trace:rootService = "gateway" }`     |
| `trace:id`               | string      | trace ID using hex string                                       | `{ trace:id = "1234567890abcde" }`      |
| `event:name`             | string      | name of event                                                   | `{ event:name = "exception" }`          |
| `event:timeSinceStart`   | duration    | time of event in relation to the span start time                | `{ event:timeSinceStart > 2ms}`         |
| `link:spanID`            | string      | link span ID using hex string                                   | `{ link:spanID = "0000000000000001" }`  |
| `link:traceID`           | string      | link trace ID using hex string                                  | `{ link:traceID = "1234567890abcde" }`  |

<!-- instrumentation scope isn't included in the 2.6 documentation
| `instrumentation:name`   | string      | instrumentation scope name                                      | `{ instrumentation:name = "grpc" }`     |
| `instrumentation:version`| string      | instrumentation scope version                                   | `{ instrumentation:version = "1.0.0" }` |
-->

The trace-level intrinsics, `trace:duration`, `trace:rootName`, and `trace:rootService`, are the same for all spans in the same trace.
Additionally, these intrinsics are significantly more performant because they have to inspect much less data then a span-level intrinsic.
They should be preferred whenever possible to span-level intrinsics.

You may have a time when you want to search by a trace-level intrinsic instead.
For example, using `span:name` looks for the names of spans within traces.
If you want to search by a trace name of `perf`, use `trace:rootName` to match against trace name.

This example searches all Kubernetes clusters called `service-name` that have a span with a root name of including `perf`.

```
{ resource.k8s.cluster.name="service-name" && trace:rootName !~ ".*perf.*"}
```

### Attribute fields

TraceQL has four different attribute scopes: span attributes, resource attributes, event attributes, and link attributes.
<!-- instrumentation scope isn't in 2.6 >
instrumentation scope attributes.  -->

By expanding a span in the Grafana UI, you can see both its span attributes (1 in the screenshot) and resource attributes (2 in the screenshot).

<p align="center"><img src="assets/span-resource-attributes.png" alt="Example of span and resource  attributes." /></p>

Attribute fields are derived from the span and can be customized.
Process and span attribute types are [defined by the attribute itself](https://github.com/open-telemetry/opentelemetry-proto/blob/b43e9b18b76abf3ee040164b55b9c355217151f3/opentelemetry/proto/common/v1/common.proto#L30-L38), whereas intrinsic fields have a built-in type.
You can refer to dynamic attributes (also known as tags) on the span or the span's resource.

Attributes in a query start with a span, resource, event, or link scope.
For example, you could use `span.http` or  `resource.namespace`, depending on what you want to query.
This provides significant performance benefits because it allows Tempo to only scan the data you are interested in.

To find traces with the `GET HTTP` method, your query could look like this:

```
{ span.http.method = "GET" }
```

For more information about attributes and resources, refer to the [OpenTelemetry Resource SDK](https://opentelemetry.io/docs/reference/specification/resource/sdk/).

#### Examples

Find traces that passed through the `production` environment:
```
{ resource.deployment.environment = "production" }
```

Find any database connection string that goes to a Postgres or MySQL database:
```
{ span.db.system =~ "postgresql|mysql" }
```

You can use the `event` scope to query events that happen within a span.
A span event is a unique point in time during the span’s duration.
While spans help build the structural hierarchy of your services, span events can provide a deeper level of granularity to help debug your application faster and maintain optimal performance.
To learn more about how you can use span events, read the [What are span events?](https://grafana.com/blog/2024/08/15/all-about-span-events-what-they-are-and-how-to-query-them/) blog post.

You can query for an exception in your span event:
```
{ event.exception.message =~ ".*something went wrong.*" }
```

If you've instrumented your traces for span links, you can use the `link` scope to query the link data. A span link associates one span with one or more other spans that are a casual relationship.
For more information on span links, refer to the [Span Links](https://opentelemetry.io/docs/concepts/signals/traces/#span-links) documentation in the Open Telemetry project.

You can search for an attribute in your link:
```
{ link.opentracing.ref_type = "child_of" }
```
<!-- instrumentation scope isn't included in the 2.6 release
Find instrumentation scope programming language:
```
{ instrumentation.language = "java" }
```
-->

### Unscoped attribute fields

Attributes can be unscoped if you are unsure if the requested attribute exists on the span or resource.
When possible, use scoped instead of unscoped attributes.
Scoped attributes provide faster query results.

For example, to find traces with an attribute of `sla` set to `critical`:
```
{ .sla = "critical" }
```

### Quoted attribute names

Attribute names can contain terminal characters, such as a period (`.`).
To search span attributes with terminal characters, you can use quoted attribute syntax.
Enclose a quoted attribute inside double quotes, for example, `"example one"`.
All characters between the quotes are considered part of the attribute name.

#### Examples

To find a span with the attribute name `attribute name with space`, use the following query:

```
{ ."attribute name with space" = "value" }
```

You can use quoted attributes syntax with non-quoted attribute syntax, the following is a valid TraceQL query:

```
{ span.attribute."attribute name with space" = "value" }
```

{{< admonition type="note" >}}
Currently, only the `\"` and `\\` escape sequences are supported.
{{< /admonition >}}

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
- `!~` (negated regular expression)

TraceQL uses Golang regular expressions.
Online regular expression testing sites like https://regex101.com/ are convenient to validate regular expressions used in TraceQL queries.
All regular expressions are treated as fully anchored.
Regular expressions are anchored at both ends. This anchoring makes the queries faster and matches the behavior of PromQL, where regular expressions are also fully anchored.

An unanchored query, such as: { span.foo =~ "bar" } is now treated as: { span.foo =~ "^bar$" }.

If you use TraceQL with regular expressions in your Grafana dashboards and you want the unanchored behavior, update the queries to use the unanchored version, such as { span.foo =~ ".*bar.*"}.

For example, to find all traces where an `http.status_code` attribute in a span are greater than `400` but less than equal to `500`:

```
{ span.http.status_code >= 400 && span.http.status_code < 500 }
```

This works for `http.status_code` values that are strings as well using lexographic ordering:

```
{ span.http.status_code >= "400" }
```

Find all traces where the `http.method` attribute is either `GET` or `DELETE`:

```
{ span.http.method =~ "DELETE|GET" }
```

Find all traces where `any_attribute` is not `nil` or where `any_attribute` exists in a span
```
{ .any_attribute != nil }
```

### Field expressions

Fields can also be combined in various ways to allow more flexible search criteria.
A field expression is a composite of multiple fields that define all of the criteria that must be matched to return results.

#### Examples

Find traces with `success` `http.status_code` codes:

```
{ span.http.status_code >= 200 && span.http.status_code < 300 }
```

Find traces where a `DELETE` HTTP method was used and the intrinsic span status wasn't OK:

```
{ span.http.method = "DELETE" && status != ok }
```

Both expressions require all conditions to be true on the same span.
The entire expression inside of a pair of `{}` must be evaluated as true on a single span for it to be included in the result set.

In the above example, if a span includes an `.http.method` attribute set to `DELETE` where the span also includes a `status` attribute set to `ok`, the trace would not be included in the returned results.

## Combine spansets

Spanset operators let you select different sets of spans from a trace and then make a determination between them.

### Logical

These spanset operators perform logical checks between the sets of spans.

- `{condA} && {condB}` - The and operator (`&&`) checks that both conditions found matches.
- `{condA} || {condB}` - The union operator (`||`) checks that either condition found matches.

For example, to find a trace that went through two specific `cloud.region`:

```
{ resource.cloud.region = "us-east-1" } && { resource.cloud.region = "us-west-1" }
```

Note the difference between the previous example and this one:

```
{ resource.cloud.region = "us-east-1" && resource.cloud.region = "us-west-1" }
```

The second expression returns no traces because it's impossible for a single span to have a `resource.cloud.region` attribute that's set to both region values at the same time.

### Structural

These spanset operators look at the structure of a trace and the relationship between the spans.
Structural operators ALWAYS return matches from the right side of the operator.

- `{condA} >> {condB}` - The descendant operator (`>>`) looks for spans matching `{condB}` that are descendants of a span matching `{condA}`
- `{condA} << {condB}` - The ancestor operator (`<<`) looks for spans matching `{condB}` that are ancestor of a span matching `{condA}`
- `{condA} > {condB}` - The child operator (`>`) looks for spans matching `{condB}` that are direct child spans of a parent matching `{condA}`
- `{condA} < {condB}` - The parent operator (`<`) looks for spans matching `{condB}` that are direct parent spans of a child matching `{condA}`
- `{condA} ~ {condB}` - The sibling operator (`~`) looks at spans matching `{condB}` that have at least one sibling matching `{condA}`.

For example, to find a trace where a specific HTTP API interacted with a specific database:

```
{ span.http.url = "/path/of/api" } >> { span.db.name = "db-shard-001" }
```

### Union structural

These spanset operators look at the structure of a trace and the relationship between the spans. These operators are unique in that they
return spans that match on both sides of the operator.

- `{condA} &>> {condB}` - The descendant operator (`>>`) looks for spans matching `{condB}` that are descendants of a span matching `{condA}`.
- `{condA} &<< {condB}` - The ancestor operator (`<<`) looks for spans matching `{condB}` that are ancestor of a span matching `{condA}`.
- `{condA} &> {condB}` - The child operator (`>`) looks for spans matching `{condB}` that are direct child spans of a parent matching `{condA}`.
- `{condA} &< {condB}` - The parent operator (`<`) looks for spans matching `{condB}` that are direct parent spans of a child matching `{condA}`.
- `{condA} &~ {condB}` - The sibling operator (`~`) looks at spans matching `{condB}` that have at least one sibling matching `{condA}`.

For example, to get a failing endpoint AND all descendant failing spans in one query:

```
{ span.http.url = "/path/of/api" && status = error } &>> { status = error }
```

### Experimental structural

These spanset operators look at the structure of a trace and the relationship between the spans.
These operators are marked experimental because sometimes return false positives.
However, the operators can be very useful (see examples below).

- `{condA} !>> {condB}` - The not-descendant operator (`!>>`) looks for spans matching `{condB}` that are not descendant spans of a parent matching `{condA}`
- `{condA} !<< {condB}` - The not-ancestor operator (`!<<`) looks for spans matching `{condB}` that are not ancestor spans of a child matching `{condA}`
- `{condA} !> {condB}` - The not-child operator (`!>`) looks for spans matching `{condB}` that are not direct child spans of a parent matching `{condA}`
- `{condA} !< {condB}` - The not-parent operator (`!<`) looks for spans matching `{condB}` that are not direct parent spans of a child matching `{condA}`
- `{condA} !~ {condB}` - The not-sibling operator (`!~`) looks that spans matching `{condB}` that do not have at least one sibling matching `{condA}`.

Read the [Tempo 2.3 blog post](/blog/2023/11/01/grafana-tempo-2.3-release-faster-trace-queries-traceql-upgrades/) for more examples and details.

For example, to find a trace with a leaf span in the service "foo":

```
{ } !< { resource.service.name = "foo" }
```

To find a span that is the last error in a series of cascading errors:

```
{ status = error } !< { status = error }
```

## Aggregators

So far, all of the example queries expressions have been about individual spans. You can use aggregate functions to ask questions about a set of spans. These currently consist of:

- `count` - The count of spans in the spanset.
- `avg` - The average of a given numeric attribute or intrinsic for a spanset.
- `max` - The max value of a given numeric attribute or intrinsic for a spanset.
- `min` - The min value of a given numeric attribute or intrinsic for a spanset.
- `sum` - The sum value of a given numeric attribute or intrinsic for a spanset.

Aggregate functions allow you to carry out operations on matching results to further refine the traces returned. For more information on planned future work, refer to [How TraceQL works]({{< relref "./architecture" >}}).

For example, to find traces where the total number of spans is greater than `10`:

```
count() > 10
```

Find traces where the average duration of the spans in a trace is greater than `20ms`:

```
avg(duration) > 20ms
```

For example, find traces that have more than 3 spans with an attribute `http.status_code` with a value of `200`:

```
{ span.http.status_code = 200 } | count() > 3
```

To find spans where the total of a made-up attribute `bytesProcessed` was more than 1 GB:

```
{ } | sum(span.bytesProcessed) > 1000000000
```

## Grouping

TraceQL supports a grouping pipeline operator that can be used to group by arbitrary attributes.
This can be useful to
find something like a single service with more than 1 error:

```
{ status = error } | by(resource.service.name) | count() > 1
```

{{< youtube id="fraepWra00Y" >}}

## Arithmetic

TraceQL supports arbitrary arithmetic in your queries. This can be useful to make queries more human readable:
```
{ span.http.request_content_length > 10 * 1024 * 1024 }
```

or anything else that comes to mind.

## Selection

TraceQL can select arbitrary fields from spans. This is particularly performant because the selected fields are not retrieved until all other criteria is met.
```
{ status=error } | select(span.http.status_code, span.http.url)
```

## Experimental TraceQL metrics

TraceQL metrics are experimental, but easy to get started with. Refer to [the TraceQL metrics]({{< relref "../operations/traceql-metrics.md" >}}) documentation for more information.
