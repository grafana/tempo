---
title: Tempo HTTP API
description: Grafana Tempo exposes an API for pushing and querying traces, and operating the cluster itself.
menuTitle: API
weight: 800
---

# Tempo HTTP API

<!-- vale Grafana.GooglePassive = NO -->
<!-- vale Grafana.Parentheses = NO -->

Tempo exposes an API for pushing and querying traces, and operating the cluster itself.

For the sake of clarity, API endpoints are grouped by service.
These endpoints are exposed both when running Tempo in microservices and monolithic mode:

- **microservices**: each service exposes its own endpoints
- **monolithic**: the Tempo process exposes all API endpoints for the services running internally

For externally supported gRPC API, [refer to Tempo gRPC API](#tempo-grpc-api).

## Endpoints

<!-- vale Grafana.Spelling = NO -->
| API | Service | Type | Endpoint |
| --- | ------- | ---- | -------- |
| [Readiness probe](#readiness-probe) | _All services_ |  HTTP | `GET /ready` |
| [Metrics](#metrics) | _All services_ |  HTTP | `GET /metrics` |
| [Pprof](#pprof) | _All services_ |  HTTP | `GET /debug/pprof` |
| [Ingest traces](#ingest) | Distributor |  - | See section for details |
| [Querying traces by id](#query) | Query-frontend |  HTTP | `GET /api/traces/<traceID>` |
| [Querying traces by id V2](#query-v2) | Query-frontend |  HTTP | `GET /api/v2/traces/<traceID>` |
| [Searching traces](#search) | Query-frontend | HTTP | `GET /api/search?<params>` |
| [Search tag names](#search-tags) | Query-frontend | HTTP | `GET /api/search/tags` |
| [Search tag names V2](#search-tags-v2) | Query-frontend | HTTP | `GET /api/v2/search/tags` |
| [Search tag values](#search-tag-values) | Query-frontend | HTTP | `GET /api/search/tag/<tag>/values` |
| [Search tag values V2](#search-tag-values-v2) | Query-frontend | HTTP | `GET /api/v2/search/tag/<tag>/values` |
| [TraceQL Metrics](#traceql-metrics) | Query-frontend | HTTP | `GET /api/metrics/query_range` |
| [TraceQL Metrics (instant)](#instant) | Query-frontend | HTTP | `GET /api/metrics/query` |
| [Query Echo Endpoint](#query-echo-endpoint) | Query-frontend |  HTTP | `GET /api/echo` |
| [Overrides API](#overrides-api) | Query-frontend | HTTP | `GET,POST,PATCH,DELETE /api/overrides` |
| Memberlist | Distributor, Ingester, Querier, Compactor |  HTTP | `GET /memberlist` |
| [Flush](#flush) | Ingester |  HTTP | `GET,POST /flush` |
| [Shutdown](#shutdown) | Ingester |  HTTP | `GET,POST /shutdown` |
| [Usage Metrics](#usage-metrics) | Distributor |  HTTP | `GET /usage_metrics` |
| [Distributor ring status](#distributor-ring-status) (*) | Distributor |  HTTP | `GET /distributor/ring` |
| [Ingesters ring status](#ingesters-ring-status) | Distributor, Querier |  HTTP | `GET /ingester/ring` |
| [Metrics-generator ring status](#metrics-generator-ring-status) (*) | Distributor |  HTTP | `GET /metrics-generator/ring` |
| [Compactor ring status](#compactor-ring-status) | Compactor |  HTTP | `GET /compactor/ring` |
| [Status](#status) | Status |  HTTP | `GET /status` |
| [List build information](#list-build-information) | Status |  HTTP | `GET /api/status/buildinfo` |

_(*) This endpoint isn't always available, check the specific section for more details._

<!-- vale Grafana.Spelling = YES -->

### Readiness probe

```
GET /ready
```

Returns status code 200 when Tempo is ready to serve traffic.

### Metrics

```
GET /metrics
```

Returns the metrics for the running Tempo service in the Prometheus exposition format.
<!-- vale Grafana.Spelling = NO -->
### Pprof


```
GET /debug/pprof/heap
GET /debug/pprof/block
GET /debug/pprof/profile
GET /debug/pprof/trace
GET /debug/pprof/goroutine
GET /debug/pprof/mutex
```

Returns the runtime profiling data in the format expected by the pprof visualization tool.
There are many things which can be profiled using this including heap, trace, goroutine, etc.

For more information, refer to the official documentation of [pprof](https://golang.org/pkg/net/http/pprof/).

<!-- vale Grafana.Spelling = YES -->

### Ingest

The Tempo distributor uses the OpenTelemetry Collector receivers as a foundation to ingest trace data.
These APIs are meant to be consumed by the corresponding client SDK or pipeline component, such as Grafana
Agent, OpenTelemetry Collector, or Jaeger Agent.

|  Protocol | Type | Docs |
|  -------- | ---- | ---- |
|  OpenTelemetry | gRPC | [Link](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/protocol/otlp.md) |
|  OpenTelemetry | HTTP | [Link](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/protocol/otlp.md) |
|  Jaeger | Thrift Compact | [Link](https://www.jaegertracing.io/docs/latest/apis/#span-reporting-apis) |
|  Jaeger | Thrift Binary | [Link](https://www.jaegertracing.io/docs/latest/apis/#span-reporting-apis) |
|  Jaeger | Thrift HTTP |  [Link](https://www.jaegertracing.io/docs/latest/apis/#span-reporting-apis) |
|  Jaeger | gRPC | [Link](https://www.jaegertracing.io/docs/latest/apis/#span-reporting-apis) |
|  Zipkin | HTTP | [Link](https://zipkin.io/zipkin-api/) |

For information on how to use the Zipkin endpoint with curl (for debugging purposes), refer to [Pushing spans with HTTP](https://grafana.com/docs/tempo/<TEMPO_VERSION>/api_docs/pushing-spans-with-http/).

### Query

The following request is used to retrieve a trace from the query frontend service in
a microservices deployment or the Tempo endpoint in a monolithic mode deployment.

```
GET /api/traces/<traceid>?start=<start>&end=<end>
```

Parameters:

- `start = (unix epoch seconds)`
  Optional. Along with `end` define a time range from which traces should be returned.
- `end = (unix epoch seconds)`
  Optional. Along with `start` define a time range from which traces should be returned. Providing both `start` and `end` includes traces for the specified time range only. If the parameters aren't provided then Tempo checks for the trace across all blocks in backend. If the parameters are provided, it only checks in the blocks within the specified time range, this can result in trace not being found or partial results if it doesn't fall in the specified time range.

The following query API is also provided on the querier service for _debugging_ purposes.

```
GET /querier/api/traces/<traceid>?mode=xxxx&blockStart=0000&blockEnd=FFFF&start=<start>&end=<end>
```

Parameters:

- `mode = (blocks|ingesters|all)`
  Specifies whether the querier should look for the trace in blocks, ingesters or both (all).
  Default = `all`
- `blockStart = (GUID)`
  Specifies the blockID start boundary. If specified, the querier only searches blocks with IDs > blockStart.
  Default = `00000000-0000-0000-0000-000000000000`
  Example: `blockStart=12345678-0000-0000-1235-000001240000`
- `blockEnd = (GUID)`
  Specifies the blockID finish boundary. If specified, the querier only searches blocks with IDs < blockEnd.
  Default = `FFFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF`
  Example: `blockStart=FFFFFFFF-FFFF-FFFF-FFFF-456787652341`
- `start = (unix epoch seconds)`
  Optional. Along with `end` define a time range from which traces should be returned.
- `end = (unix epoch seconds)`
  Optional. Along with `start` define a time range from which traces should be returned. Providing both `start` and `end` includes blocks for the specified time range only.

This API isn't meant to be used directly unless for debugging the sharding functionality of the query
frontend.

**Returns**

By default, this endpoint returns a mostly compatible [OpenTelemetry](https://github.com/open-telemetry/opentelemetry-proto/tree/main/opentelemetry/proto/trace/v1) JSON,
but if it can also send OpenTelemetry proto if `Accept: application/protobuf` is passed.


### Query V2

The following request is used to retrieve a trace from the query frontend service in
a microservices deployment or the Tempo endpoint in a monolithic mode deployment.

```
GET /api/v2/traces/<traceid>?start=<start>&end=<end>
```

Parameters:

- `start = (unix epoch seconds)`
  Optional. Along with `end` define a time range from which traces should be returned.
- `end = (unix epoch seconds)`
  Optional. Along with `start` define a time range from which traces should be returned. Providing both `start` and `end` includes traces for the specified time range only. If the parameters aren't provided then Tempo checks for the trace across all blocks in backend. If the parameters are provided, it only checks in the blocks within the specified time range, this can result in trace not being found or partial results if it doesn't fall in the specified time range.

The following query API is also provided on the querier service for _debugging_ purposes.

```
GET /querier/api/v2/traces/<traceid>?mode=xxxx&blockStart=0000&blockEnd=FFFF&start=<start>&end=<end>
```

Parameters:

- `mode = (blocks|ingesters|all)`
  Specifies whether the querier should look for the trace in blocks, ingesters or both (all).
  Default = `all`
- `blockStart = (GUID)`
  Specifies the blockID start boundary. If specified, the querier only searches blocks with IDs > blockStart.
  Default = `00000000-0000-0000-0000-000000000000`
  Example: `blockStart=12345678-0000-0000-1235-000001240000`
- `blockEnd = (GUID)`
  Specifies the blockID finish boundary. If specified, the querier only searches blocks with IDs < blockEnd.
  Default = `FFFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF`
  Example: `blockStart=FFFFFFFF-FFFF-FFFF-FFFF-456787652341`
- `start = (unix epoch seconds)`
  Optional. Along with `end` define a time range from which traces should be returned.
- `end = (unix epoch seconds)`
  Optional. Along with `start` define a time range from which traces should be returned. Providing both `start` and `end` includes blocks for the specified time range only.

**Returns**

By default, this endpoint returns Query response with a [OpenTelemetry](https://github.com/open-telemetry/opentelemetry-proto/tree/main/opentelemetry/proto/trace/v1) JSON trace,
but if it can also send OpenTelemetry proto if `Accept: application/protobuf` is passed.

### Search

The Tempo Search API finds traces based on span and process attributes (tags and values). Note that search functionality is **not** available on
[v2 blocks](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/parquet/#choose-a-different-block-format).

When performing a search, Tempo does a massively parallel search over the given time range, and takes the first N results. Even identical searches differs due to things like machine load and network latency. TraceQL follows the same behavior.

The API is available in the query frontend service in
a microservices deployment, or the Tempo endpoint in a monolithic mode deployment.

The following request is used to find traces containing spans from service `myservice` and the URL contains `api/myapi`.

```
GET /api/search?tags=service.name%3Dmyservice%20http.url%3Dapi%2Fmyapi
```

The URL query parameters support the following values:

**Parameters for TraceQL Search**

- `q = (TraceQL query)`: URL encoded [TraceQL query](../traceql).

**Parameters for Tag Based Search**

- `tags = (logfmt)`: `logfmt` encoding of any span-level or process-level attributes to filter on. The value is matched as a case-insensitive substring. Key-value pairs are separated by spaces. If a value contains a space, it should be enclosed within double quotes.
- `minDuration = (go duration value)`
  Optional. Find traces with at least this duration. Duration values are of the form `10s` for 10 seconds, `100ms`, `30m`, etc.
- `maxDuration = (go duration value)`
  Optional. Find traces with no greater than this duration. Uses the same form as `minDuration`.

**Parameters supported for all searches**

- `limit = (integer)`
  Optional. Limit the number of search results. Default is 20, but this is configurable in the querier. Refer to [Configuration](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#querier).
- `start = (unix epoch seconds)`
  Optional. Along with `end` define a time range from which traces should be returned.
- `end = (unix epoch seconds)`
 Optional. Along with `start`, define a time range from which traces should be returned. Providing both `start` and `end` changes the way that Tempo searches.
 If the parameters aren't provided, then Tempo searches the recent trace data stored in the ingesters. If the parameters are provided, it searches the backend as well.
 - `spss = (integer)`
  Optional. Limit the number of spans per span-set. Default value is 3.

#### Example of TraceQL search

Example of how to query Tempo using curl.
This query returns all traces that have their status set to error.

```bash
curl -G -s http://localhost:3200/api/search --data-urlencode 'q={ status=error }' | jq
{
  "traces": [
    {
      "traceID": "2f3e0cee77ae5dc9c17ade3689eb2e54",
      "rootServiceName": "shop-backend",
      "rootTraceName": "update-billing",
      "startTimeUnixNano": "1684778327699392724",
      "durationMs": 557,
      "spanSets": [
        {
          "spans": [
            {
              "spanID": "563d623c76514f8e",
              "startTimeUnixNano": "1684778327735077898",
              "durationNanos": "446979497",
              "attributes": [
                {
                  "key": "status",
                  "value": {
                    "stringValue": "error"
                  }
                }
              ]
            }
          ],
          "matched": 1
        }
      ]
  ],
  "metrics": {
    "totalBlocks": 13
  }
}
```

#### Example of tags-based search

Example of how to query Tempo using curl.
This query returns all traces that have a tag `service.name` containing `cartservice` and a minimum duration of 600 ms.

```bash
curl -G -s http://localhost:3200/api/search --data-urlencode 'tags=service.name=cartservice' --data-urlencode minDuration=600ms | jq
{
  "traces": [
    {
      "traceID": "d6e9329d67b6146a",
      "rootServiceName": "frontend",
      "rootTraceName": "/cart",
      "startTimeUnixNano": "1634727903545000000",
      "durationMs": 611
    },
    {
      "traceID": "1b1ba462b409200d",
      "rootServiceName": "frontend",
      "rootTraceName": "/cart",
      "startTimeUnixNano": "1634727775935000000",
      "durationMs": 611
    }
  ],
  "metrics": {
    "inspectedTraces": 3100,
    "inspectedBytes": "3811736",
    "totalBlocks": 3
  }
}
```

### Search tags

Ingester configuration `complete_block_timeout` affects how long tags are available for search.

This endpoint retrieves all discovered tag names that can be used in search.
The endpoint is available in the query frontend service in a microservices deployment, or the Tempo endpoint in a monolithic mode deployment.
The tags endpoint takes a scope that controls the kinds
of tags or attributes returned. If nothing is provided, the endpoint returns all resource and span tags.

```
GET /api/search/tags?scope=<resource|span|intrinsic>
```

#### Example

Example of how to query Tempo using curl.
This query returns all discovered tag names.

```bash
curl -G -s http://localhost:3200/api/search/tags?scope=span  | jq
{
  "tagNames": [
    "host.name",
    "http.method",
    "http.status_code",
    "http.url",
    "ip",
    "load_generator.seq_num",
    "name",
    "region",
    "root_cause_error",
    "sampler.param",
    "sampler.type",
    "service.name",
    "starter",
    "version"
  ]
  "metrics": {
    "inspectedBytes": "630188"
  }
}
```

Parameters:

- `scope = (resource|span|intrinsic)`
  Optional. Specifies the scope of the tags. If not specified, it means all scopes.
  Default = `all`
- `start = (unix epoch seconds)`
  Optional. Along with `end`, defines a time range from which tags should be returned.
- `end = (unix epoch seconds)`
  Optional. Along with `start`, defines a time range from which tags should be returned. Providing both `start` and `end` includes blocks for the specified time range only.
- `limit = (integer)`
  Optional. Limits the maximum number of tags values.
- `maxStaleValues = (integer)`
  Optional. Limits the search for tags names. If the number of stale (already known) values reaches or exceeds this limit, the search stops. If Tempo processes `maxStaleValues` matches without finding a new tag name, the search is returned early.

### Search tags V2

Ingester configuration `complete_block_timeout` affects how long tags are available for search.
If the start or end aren't specified, it only fetches blocks that weren't flushed to backend.

This endpoint retrieves all discovered tag names that can be used in search.
The endpoint is available in the query frontend service in
a microservices deployment, or the Tempo endpoint in a monolithic mode deployment.
The tags endpoint takes a scope that controls the kinds of tags or attributes returned.
If nothing is provided, the endpoint returns all resource and span tags.

```bash
GET /api/v2/search/tags?scope=<resource|span|intrinsic>
```

Parameters:

- `scope = (resource|span|intrinsic)`
  Specifies the scope of the tags, this is an optional parameter, if not specified it means all scopes.
  Default = `all`
- `q = (traceql query)`
  Optional. A TraceQL query to filter tag names by. Currently only works for a single spanset of `&&`ed conditions. For example: `{ span.foo = "bar" && resource.baz = "bat" ...}`. See also [Filtered tag values](#filtered-tag-values).
- `start = (unix epoch seconds)`
  Optional. Along with `end` define a time range from which tags should be returned.
- `end = (unix epoch seconds)`
  Optional. Along with `start` define a time range from which tags should be returned. Providing both `start` and `end` includes blocks for the specified time range only.
- `limit = (integer)`
  Optional. Sets the maximum number of tags names allowed per scope. The query stops once this limit is reached for any scope.
- `maxStaleValues = (integer)`
  Optional. Limits the search for tag values. The search stops if the number of stale (already known) values reaches or exceeds this limit.

#### Example

Example of how to query Tempo using curl.
This query returns all discovered tag names.

```bash
curl -G -s http://localhost:3200/api/v2/search/tags  | jq
{
  "scopes": [
    {
      "name": "link",
      "tags": [
        "link-type"
      ]
    },
    {
      "name": "resource",
      "tags": [
        "k6",
        "service.name"
      ]
    },
    {
      "name": "span",
      "tags": [
        "article.count",
        "http.flavor",
        "http.method",
        "http.request.header.accept",
        "http.request_content_length",
        "http.response.header.content-type",
        "http.response_content_length",
        "http.scheme",
        "http.status_code",
        "http.target",
        "http.url",
        "net.host.name",
        "net.host.port",
        "net.peer.name",
        "net.peer.port",
        "net.sock.family",
        "net.sock.host.addr",
        "net.sock.peer.addr",
        "net.transport",
        "numbers",
        "one"
      ]
    },
    {
      "name": "intrinsic",
      "tags": [
        "duration",
        "event:name",
        "event:timeSinceStart",
        "instrumentation:name",
        "instrumentation:version",
        "kind",
        "name",
        "rootName",
        "rootServiceName",
        "span:duration",
        "span:kind",
        "span:name",
        "span:status",
        "span:statusMessage",
        "status",
        "statusMessage",
        "trace:duration",
        "trace:rootName",
        "trace:rootService",
        "traceDuration"
      ]
    },
    {
      "name": "event",
      "tags": [
        "exception.escape",
        "exception.message",
        "exception.stacktrace",
        "exception.type",
      ]
    }
  ],
  "metrics": {
    "inspectedBytes": "377046"
  }
}
```

### Search tag values

Ingester configuration `complete_block_timeout` affects how long tags are available for search.
If start or end aren't specified, it only fetches blocks that wasn't flushed to backend.

This endpoint retrieves all discovered values for the given tag, which can be used in search.
The endpoint is available in the query frontend service in a microservices deployment, or the Tempo endpoint in a monolithic mode deployment.
The following request returns all discovered service names.

```bash
GET /api/search/tag/service.name/values
```

#### Example

Example of how to query Tempo using curl.
This query returns all discovered values for the tag `service.name`.

```bash
curl -G -s http://localhost:3200/api/search/tag/service.name/values  | jq
{
  "tagValues": [
    "article-service",
    "auth-service",
    "billing-service",
    "cart-service",
    "postgres",
    "shop-backend"
  ],
  "metrics": {
    "inspectedBytes": "431380"
  }
}
```

Parameters:
- `start = (unix epoch seconds)`
  Optional. Along with `end`, defines a time range from which tags should be returned.
- `end = (unix epoch seconds)`
  Optional. Along with `start`, defines a time range from which tags should be returned. Providing both `start` and `end` includes blocks for the specified time range only.
- `limit = (integer)`
  Optional. Limits the maximum number of tags values.
- `maxStaleValues = (integer)`
  Optional. Limits the search for tags values. If the number of stale (already known) values reaches or exceeds this limit, the search stops. If Tempo processes `maxStaleValues` matches without finding a new tag name, the search is returned early.


### Search tag values V2

This endpoint retrieves all discovered values and their data types for the given TraceQL identifier.
The endpoint is available in the query frontend service in a microservices deployment, or the Tempo endpoint in a monolithic mode deployment. This endpoint is similar to `/api/search/tag/<tag>/values` but operates on TraceQL identifiers and types.
Refer to [TraceQL](../traceql) documentation for more information.

#### Example

This example queries Tempo using curl and returns all discovered values for the tag `service.name`.

```bash
curl -G -s http://localhost:3200/api/v2/search/tag/.service.name/values | jq
{
  "tagValues": [
    {
      "type": "string",
      "value": "article-service"
    },
    {
      "type": "string",
      "value": "postgres"
    },
    {
      "type": "string",
      "value": "cart-service"
    },
    {
      "type": "string",
      "value": "billing-service"
    },
    {
      "type": "string",
      "value": "shop-backend"
    },
    {
      "type": "string",
      "value": "auth-service"
    }
  ],
  "metrics": {
    "inspectedBytes": "502756"
  }
}
```

Parameters:
- `start = (unix epoch seconds)`
  Optional. Along with `end`, defines a time range from which tags values should be returned.
- `end = (unix epoch seconds)`
  Optional. Along with `start`, defines a time range from which tags values should be returned. Providing both `start` and `end` includes blocks for the specified time range only.
- `q = (traceql query)`
  Optional. A TraceQL query to filter tag values by. Currently only works for a single spanset of `&&`ed conditions. For example: `{ span.foo = "bar" && resource.baz = "bat" ...}`. Refer to [Filtered tag values](#filtered-tag-values).
- `limit = (integer)`
  Optional. Limits the maximum number of tags values
- `maxStaleValues = (integer)`
  Optional. Limits the search for tags values. If the number of stale (already known) values reaches or exceeds this limit, the search stops. If Tempo processes `maxStaleValues` matches without finding a new tag name, the search is returned early.

#### Filtered tag values

You can pass an optional URL query parameter, `q`, to your request.
The `q` parameter is a URL-encoded [TraceQL query](../traceql).
If provided, the tag values returned by the API are filtered to only return values seen on spans matching your filter parameters.

Queries can be incomplete: for example, `{ resource.cluster = }`.
Tempo extracts only the valid matchers and builds a valid query.
If an input is invalid, Tempo doesn't provide an error. Instead,
you'll see the whole list when a failure of parsing input. This behavior helps with backwards compatibility.

Only queries with a single selector `{}` and AND `&&` operators are supported.
  - Example supported: `{ resource.cluster = "us-east-1" && resource.service = "frontend" }`
  - Example unsupported: `{ resource.cluster = "us-east-1" || resource.service = "frontend" } && { resource.cluster = "us-east-2" }`

Unscoped attributes aren't supported for filtered tag values.

The following request returns all discovered service names on spans with `span.http.method=GET`:

```
GET /api/v2/search/tag/resource.service.name/values?q="{span.http.method='GET'}"
```

If a particular service name (for example, `shopping-cart`) is only present on spans with `span.http.method=POST`, it won't be included in the list of values returned.

### TraceQL Metrics

The TraceQL Metrics API returns Prometheus-like time-series for a given metrics query.
Metrics queries are those using metrics functions like `rate()` and `quantile_over_time()`.
Refer to the [TraceQL metrics documentation](https://grafana.com/docs/tempo/<TEMPO_VERSION>/traceql/metrics-queries/) for more information list.

Parameters:

- `q = (traceql query)`
  The TraceQL metrics query to process.
- `start = (unix epoch seconds | unix epoch nanoseconds | RFC3339 string)`
  Optional. Along with `end` defines the time range.
- `end = (unix epoch seconds | unix epoch nanoseconds | RFC3339 string)`
  Optional. Along with `start` define the time range. Providing both `start` and `end` includes blocks for the specified time range only.
- `since = (duration string)`
  Optional. Can be used instead of `start` and `end` to define the time range in relative values. For example, `since=15m` queries the last 15 minutes. Default is the last 1 hour.
- `step = (duration string)`
  Optional. Defines the granularity of the returned time-series. For example, `step=15s` returns a data point every 15s within the time range. If not specified, then the default behavior chooses a dynamic step based on the time range.
- `exemplars = (integer)`
  Optional. Defines the maximum number of exemplars for the query. It's trimmed to `max_exemplars` if it exceeds it.

The API is available in the query frontend service in
a microservices deployment, or the Tempo endpoint in a monolithic mode deployment.

For example, the following request computes the rate of spans received for `myservice` over the last three hours, at 1 minute intervals.

{{< admonition type="note" >}}
Actual API parameters must be URL-encoded. This example is left unencoded for readability.
{{< /admonition >}}

```
GET /api/metrics/query_range?q={resource.service.name="myservice"} | min_over_time() with(exemplars=true) &since=3h&step=1m&exemplars=100
```

#### Instant

The instant version of the metrics API is similar to the range version, but instead returns a single value for the query.
This version is useful when you don't need the granularity of a full time-series, but instead want a total sum, or single value computed across the whole time range.

The parameters are identical to the range version except there is no `step`.

Parameters:

- `q = (traceql query)`
  The TraceQL metrics query to process.
- `start = (unix epoch seconds | unix epoch nanoseconds | RFC3339 string)`
  Optional. Along with `end` defines the time range.
- `end = (unix epoch seconds | unix epoch nanoseconds | RFC3339 string)`
  Optional. Along with `start` define the time range. Providing both `start` and `end` includes blocks for the specified time range only.
- `since = (duration string)`
  Optional. Can be used instead of `start` and `end` to define the time range in relative values. For example, `since=15m` queries the last 15 minutes. Default is last 1 hour.

The API is available in the query frontend service in
a microservices deployment, or the Tempo endpoint in a monolithic mode deployment.

For example the following request computes the total number of failed spans over the last hour per service.

{{< admonition type="note" >}}
Actual API parameters must be URL-encoded. This example is left unencoded for readability.
{{< /admonition >}}

```
GET /api/metrics/query?q={status=error}|count_over_time()by(resource.service.name)
```

### Query Echo endpoint

```
GET /api/echo
```

Returns status code 200 and body `echo` when the query frontend is up and ready to receive requests.

{{< admonition type="note" >}}
Meant to be used in a Query Visualization UI like Grafana to test that the Tempo data source is working.
{{< /admonition >}}

### Overrides API

For more information about user-configurable overrides API, refer to the [user-configurable overrides](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/manage-advanced-systems/user-configurable-overrides/#api) documentation.

### Flush

```
GET,POST /flush
```

Triggers a flush of all in-memory traces to the WAL. Useful at the time of rollout restarts and unexpected crashes.

Specify the `tenant` parameter to flush data of a single tenant only.

```
GET,POST /flush?tenant=dev
```

### Shutdown

```
GET,POST /shutdown
```

Flushes all in-memory traces and the WAL to the long term backend. Gracefully exits from the ring. Shuts down the
ingester service.

{{< admonition type="note" >}}
This is usually used at the time of scaling down a cluster.
{{< /admonition >}}

### Usage metrics

{{< admonition type="note" >}}
This endpoint is only available when one or more usage trackers are enabled in [the distributor](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#distributor).
{{< /admonition >}}

```
GET /usage_metrics
```

Special metrics scrape endpoint that provides per-tenant metrics on ingested data. Per-tenant grouping rules are configured in [the per-tenant overrides](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#overrides)

Example:
```
curl http://localhost:3200/usage_metrics
# HELP tempo_usage_tracker_bytes_received_total bytes total received with these attributes
# TYPE tempo_usage_tracker_bytes_received_total counter
tempo_usage_tracker_bytes_received_total{service="auth-service",tenant="single-tenant",tracker="cost-attribution"} 96563
tempo_usage_tracker_bytes_received_total{service="cache",tenant="single-tenant",tracker="cost-attribution"} 81904
tempo_usage_tracker_bytes_received_total{service="gateway",tenant="single-tenant",tracker="cost-attribution"} 164751
tempo_usage_tracker_bytes_received_total{service="identity-service",tenant="single-tenant",tracker="cost-attribution"} 85974
tempo_usage_tracker_bytes_received_total{service="service-A",tenant="single-tenant",tracker="cost-attribution"} 92799
```

### Distributor ring status

{{< admonition type="note" >}}
This endpoint is only available when Tempo is configured with [the global override strategy](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#overrides).
{{< /admonition >}}

```
GET /distributor/ring
```

Displays a web page with the distributor hash ring status, including the state, healthy, and last heartbeat time of each
distributor.

For more information, refer to [consistent hash ring](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/manage-advanced-systems/consistent_hash_ring/).

### Ingesters ring status

```
GET /ingester/ring
```

Displays a web page with the ingesters hash ring status, including the state, healthy, and last heartbeat time of each ingester.

For more information, refer to [consistent hash ring](http://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/manage-advanced-systems/consistent_hash_ring/).

### Metrics-generator ring status

```
GET /metrics-generator/ring
```

Displays a web page with the metrics-generator hash ring status, including the state, health, and last heartbeat time of each metrics-generator.

This endpoint is only available when the metrics-generator is enabled. Refer to [metrics-generator](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#metrics-generator).

For more information, refer to [consistent hash ring](http://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/manage-advanced-systems/consistent_hash_ring/).

### Compactor ring status

```
GET /compactor/ring
```

Displays a web page with the compactor hash ring status, including the state, healthy, and last heartbeat time of each compactor.

For more information, refer to [consistent hash ring](http://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/manage-advanced-systems/consistent_hash_ring/).

### Status

```
GET /status
```
Print all available information by default.

```
GET /status/version
```

Print the version information.

```
GET /status/services
```

Displays a list of services and their status. If a service failed it shows the failure case.

```
GET /status/endpoints
```

Displays status information about the API endpoints.

```
GET /status/config
```

Displays the configuration.

Displays the configuration currently applied to Tempo (in YAML format), including default values and settings via CLI flags.
Sensitive data is masked. Be aware that the exported configuration **doesn't include the per-tenant overrides**.

Optional query parameter:

- `mode = (diff|defaults)`: `diff` shows the difference between the default values and the current configuration. `defaults` shows the default values.

```
GET /status/runtime_config
```

Displays the override configuration.

Query parameter:

- `mode = (diff)`: Show the difference between defaults and overrides.

```
GET /status/overrides
```

Displays all tenants that have non-default overrides configured.

```
GET /status/overrides/{tenant}
```

Displays all overrides configured for the specified tenant.

```
GET /status/usage-stats
```

Displays anonymous usage stats data that's reported back to Grafana Labs.

### List build information

```
GET /api/status/buildinfo
```
Exposes the build information in a JSON object. The fields are `version`, `revision`, `branch`, `buildDate`, `buildUser`, and `goVersion`.

## Tempo gRPC API

Tempo uses [gRPC](https://grpc.io) to internally communicate with itself, but only has one externally supported client.
The query-frontend component implements the streaming querier interface defined below.
[Refer here](https://github.com/grafana/tempo/blob/main/pkg/tempopb/) for the complete proto definition and generated code.

By default, this service is only offered over the gRPC port.
You can use streaming service over the HTTP port as well, which Grafana expects.

To enable the streaming service over the HTTP port for use with Grafana, set the following:

```yaml
stream_over_http_enabled: true
```

The query frontend supports the following interface. Refer to [`tempo.proto`](https://github.com/grafana/tempo/blob/main/pkg/tempopb/tempo.proto) for complete details of all objects.

```protobuf
service StreamingQuerier {
  rpc Search(SearchRequest) returns (stream SearchResponse);
  rpc SearchTags(SearchTagsRequest) returns (stream SearchTagsResponse) {}
  rpc SearchTagsV2(SearchTagsRequest) returns (stream SearchTagsV2Response) {}
  rpc SearchTagValues(SearchTagValuesRequest) returns (stream SearchTagValuesResponse) {}
  rpc SearchTagValuesV2(SearchTagValuesRequest) returns (stream SearchTagValuesV2Response) {}
  rpc MetricsQueryRange(QueryRangeRequest) returns (stream QueryRangeResponse) {}
}
```

{{< admonition type="note" >}}
gRPC compression is disabled by default.
Refer to [gRPC compression configuration](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#grpc-compression) for more information.
{{< /admonition >}}
<!-- vale Grafana.GooglePassive = YES -->