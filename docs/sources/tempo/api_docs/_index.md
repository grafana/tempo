---
title: Tempo API
description: Grafana Tempo exposes an API for pushing and querying traces, and operating the cluster itself.
menuTitle: Tempo API
weight: 500
---

# Tempo API

Tempo exposes an API for pushing and querying traces, and operating the cluster itself.

For the sake of clarity, API endpoints are grouped by service.
These endpoints are exposed both when running Tempo in microservices and monolithic mode:
- **microservices**: each service exposes its own endpoints
- **monolithic**: the Tempo process exposes all API endpoints for the services running internally

## Endpoints

| API | Service | Type | Endpoint |
| --- | ------- | ---- | -------- |
| [Readiness probe](#readiness-probe) | _All services_ |  HTTP | `GET /ready` |
| [Metrics](#metrics) | _All services_ |  HTTP | `GET /metrics` |
| [Pprof](#pprof) | _All services_ |  HTTP | `GET /debug/pprof` |
| [Ingest traces](#ingest) | Distributor |  - | See section for details |
| [Querying traces by id](#query) | Query-frontend |  HTTP | `GET /api/traces/<traceID>` |
| [Searching traces](#search) | Query-frontend | HTTP | `GET /api/search?<params>` |
| [Search tag names](#search-tags) | Query-frontend | HTTP | `GET /api/search/tags` |
| [Search tag values](#search-tag-values) | Query-frontend | HTTP | `GET /api/search/tag/<tag>/values` |
| [Search tag values V2](#search-tag-values-v2) | Query-frontend | HTTP | `GET /api/v2/search/tag/<tag>/values` |
| [Query Echo Endpoint](#query-echo-endpoint) | Query-frontend |  HTTP | `GET /api/echo` |
| Memberlist | Distributor, Ingester, Querier, Compactor |  HTTP | `GET /memberlist` |
| [Flush](#flush) | Ingester |  HTTP | `GET,POST /flush` |
| [Shutdown](#shutdown) | Ingester |  HTTP | `GET,POST /shutdown` |
| [Distributor ring status](#distributor-ring-status) (*) | Distributor |  HTTP | `GET /distributor/ring` |
| [Ingesters ring status](#ingesters-ring-status) | Distributor, Querier |  HTTP | `GET /ingester/ring` |
| [Metrics-generator ring status](#metrics-generator-ring-status) (*) | Distributor |  HTTP | `GET /metrics-generator/ring` |
| [Compactor ring status](#compactor-ring-status) | Compactor |  HTTP | `GET /compactor/ring` |
| [Status](#status) | Status |  HTTP | `GET /status` |

_(*) This endpoint is not always available, check the specific section for more details._

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

_For more information, please check out the official documentation of [pprof](https://golang.org/pkg/net/http/pprof/)._

### Ingest

The Tempo distributor uses the OpenTelemetry Collector receivers as a foundation to ingest trace data.
These APIs are meant to be consumed by the corresponding client SDK or pipeline component, such as Grafana
Agent, OpenTelemetry Collector, or Jaeger Agent.

|  Protocol | Type | Docs |
|  -------- | ---- | ---- |
|  OpenTelemetry | GRPC | [Link](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/protocol/otlp.md) |
|  OpenTelemetry | HTTP | [Link](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/protocol/otlp.md) |
|  Jaeger | Thrift Compact | [Link](https://www.jaegertracing.io/docs/latest/apis/#span-reporting-apis) |
|  Jaeger | Thrift Binary | [Link](https://www.jaegertracing.io/docs/latest/apis/#span-reporting-apis) |
|  Jaeger | Thrift HTTP |  [Link](https://www.jaegertracing.io/docs/latest/apis/#span-reporting-apis) |
|  Jaeger | GRPC | [Link](https://www.jaegertracing.io/docs/latest/apis/#span-reporting-apis) |
|  Zipkin | HTTP | [Link](https://zipkin.io/zipkin-api/) |

For information on how to use the Zipkin endpoint with curl (for debugging purposes), refer to [Pushing spans with HTTP]({{< relref "pushing-spans-with-http/" >}}).

### Query

The following request is used to retrieve a trace from the query frontend service in
a microservices deployment or the Tempo endpoint in a monolithic mode deployment.

```
GET /api/traces/<traceid>?start=<start>&end=<end>
```
Parameters:
- `start = (unix epoch seconds)`
  Optional.  Along with `end` define a time range from which traces should be returned.
- `end = (unix epoch seconds)`
  Optional.  Along with `start` define a time range from which traces should be returned. Providing both `start` and `end` will include traces for the specified time range only. If the parameters are not provided then Tempo will check for the trace across all blocks in backend. If the parameters are provided, it will only check in the blocks within the specified time range, this can result in trace not being found or partial results if it does not fall in the specified time range.

The following query API is also provided on the querier service for _debugging_ purposes.

```
GET /querier/api/traces/<traceid>?mode=xxxx&blockStart=0000&blockEnd=FFFF&start=<start>&end=<end>
```
Parameters:
- `mode = (blocks|ingesters|all)`
  Specifies whether the querier should look for the trace in blocks, ingesters or both (all).
  Default = `all`
- `blockStart = (GUID)`
  Specifies the blockID start boundary. If specified, the querier will only search blocks with IDs > blockStart.
  Default = `00000000-0000-0000-0000-000000000000`
  Example: `blockStart=12345678-0000-0000-1235-000001240000`
- `blockEnd = (GUID)`
  Specifies the blockID finish boundary. If specified, the querier will only search blocks with IDs < blockEnd.
  Default = `FFFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF`
  Example: `blockStart=FFFFFFFF-FFFF-FFFF-FFFF-456787652341`
- `start = (unix epoch seconds)`
  Optional.  Along with `end` define a time range from which traces should be returned.
- `end = (unix epoch seconds)`
  Optional.  Along with `start` define a time range from which traces should be returned. Providing both `start` and `end` will include blocks for the specified time range only.

This API is not meant to be used directly unless for debugging the sharding functionality of the query
frontend.

Returns:
By default this endpoint returns [OpenTelemetry](https://github.com/open-telemetry/opentelemetry-proto/tree/main/opentelemetry/proto/trace/v1) JSON,
but if it can also send OpenTelemetry proto if `Accept: application/protobuf` is passed.

### Search

Tempo's Search API finds traces based on span and process attributes (tags and values). Note that search functionality is **not** available on
[v2 blocks]({{< relref "../configuration/parquet#disable-parquet" >}}).

When performing a search, Tempo does a massively parallel search over the given time range, and takes the first N results. Even identical searches will differ due to things like machine load and network latency. TraceQL follows the same behavior.

The API is available in the query frontend service in
a microservices deployment, or the Tempo endpoint in a monolithic mode deployment.

The following request is used to find traces containing spans from service `myservice` and the url contains `api/myapi`.

```
GET /api/search?tags=service.name%3Dmyservice%20http.url%3Dapi%2Fmyapi
```

The URL query parameters support the following values:

**Parameters for TraceQL Search**
- `q = (TraceQL query)`: Url encoded [TraceQL query](https://grafana.com/docs/tempo/latest/traceql/).

**Parameters for Tag Based Search**
- `tags = (logfmt)`: logfmt encoding of any span-level or process-level attributes to filter on. The value is matched as a case-insensitive substring. Key-value pairs are separated by spaces. If a value contains a space, it should be enclosed within double quotes.
- `minDuration = (go duration value)`
  Optional.  Find traces with at least this duration.  Duration values are of the form `10s` for 10 seconds, `100ms`, `30m`, etc.
- `maxDuration = (go duration value)`
  Optional.  Find traces with no greater than this duration.  Uses the same form as `minDuration`.

**Parameters supported for all searches**
- `limit = (integer)`
  Optional.  Limit the number of search results. Default is 20, but this is configurable in the querier. Refer to [Configuration]({{< relref "../configuration#querier" >}}).
- `start = (unix epoch seconds)`
  Optional.  Along with `end` define a time range from which traces should be returned.
- `end = (unix epoch seconds)`
 Optional.  Along with `start`, define a time range from which traces should be returned. Providing both `start` and `end` will change the way that Tempo searches.
 If the parameters are not provided, then Tempo will search the recent trace data stored in the ingesters. If the parameters are provided, it will search the backend as well.

#### Example of TraceQL search

Example of how to query Tempo using curl.
This query will return all traces that have their status set to error.

```bash
$ curl -G -s http://localhost:3200/api/search --data-urlencode 'q={ status=error }' | jq
{
  "traces": [
    {
      "traceID": "169bdefcae1f19",
      "rootServiceName": "gme-ruler",
      "rootTraceName": "rule",
      "startTimeUnixNano": "1675090379953800000",
      "durationMs": 3,
      "spanSet": {
        "spans": [
          {
            "spanID": "45b795d0c4f9f6ae",
            "startTimeUnixNano": "1675090379955688000",
            "durationNanos": "525000",
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
    },
  ],
  "metrics": {
    "inspectedBlocks": 13
  }
}
```

#### Example of Tags Based Search

Example of how to query Tempo using curl.
This query will return all traces that have a tag `service.name` containing `cartservice` and a minimum duration of 600 ms.

```bash
$ curl -G -s http://localhost:3200/api/search --data-urlencode 'tags=service.name=cartservice' --data-urlencode minDuration=600ms | jq
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
    "inspectedBlocks": 3
  }
}
```

### Search tags

Ingester configuration `complete_block_timeout` affects how long tags are available for search.

This endpoint retrieves all discovered tag names that can be used in search.  The endpoint is available in the query frontend service in
a microservices deployment, or the Tempo endpoint in a monolithic mode deployment.

```
GET /api/search/tags
```

#### Example

Example of how to query Tempo using curl.
This query will return all discovered tag names.

```bash
$ curl -G -s http://localhost:3200/api/search/tags  | jq
{
  "tagNames": [
    "host.name",
    "http.method",
    "http.status_code",
    "http.url",
    "ip",
    "load_generator.seq_num",
    "name",
    "opencensus.exporterversion",
    "region",
    "root.name",
    "root.service.name",
    "root_cause_error",
    "sampler.param",
    "sampler.type",
    "service.name",
    "starter",
    "version"
  ]
}
```

### Search tag values

Ingester configuration `complete_block_timeout` affects how long tags are available for search.

This endpoint retrieves all discovered values for the given tag, which can be used in search.  The endpoint is available in the query frontend service in
a microservices deployment, or the Tempo endpoint in a monolithic mode deployment.  The following request will return all discovered service names.

```
GET /api/search/tag/service.name/values
```

#### Example

Example of how to query Tempo using curl.
This query will return all discovered values for the tag `service.name`.

```bash
$ curl -G -s http://localhost:3200/api/search/tag/service.name/values  | jq
{
  "tagValues": [
    "adservice",
    "cartservice",
    "checkoutservice",
    "frontend",
    "productcatalogservice",
    "recommendationservice"
  ]
}
```

### Search tag values V2

This endpoint retrieves all discovered values and their data types for the given TraceQL identifier.  The endpoint is available in the query frontend service in
a microservices deployment, or the Tempo endpoint in a monolithic mode deployment. This endpoint is similar to `/api/search/tag/<tag>/values` but operates on TraceQL identifiers and types. See [TraceQL](../traceql/) documention for more information. The following request returns all discovered service names.

```
GET /api/v2/search/tag/.service.name/values
```

#### Example

This example queries Tempo using curl and returns all discovered values for the tag `service.name`.

```bash
$ curl http://localhost:3200/api/v2/search/tag/.service.name/values | jq .
{
  "tagValues": [
    {
      "type": "string",
      "value": "customer"
    },
    {
      "type": "string",
      "value": "mysql"
    },
    {
      "type": "string",
      "value": "driver"
    },
    {
      "type": "string",
      "value": "frontend"
    },
    {
      "type": "string",
      "value": "redis"
    }
  ]
}
```

### Query Echo Endpoint

```
GET /api/echo
```

Returns status code 200 and body `echo` when the query frontend is up and ready to receive requests.

**Note**: Meant to be used in a Query Visualization UI like Grafana to test that the Tempo datasource is working.


### Flush

```
GET,POST /flush
```

Triggers a flush of all in-memory traces to the WAL. Useful at the time of rollout restarts and unexpected crashes.

### Shutdown

```
GET,POST /shutdown
```

Flushes all in-memory traces and the WAL to the long term backend. Gracefully exits from the ring. Shuts down the
ingester service.

**Note**: This is usually used at the time of scaling down a cluster.

### Distributor ring status

> **Note**: This endpoint is only available when Tempo is configured with [the global override strategy]({{< relref "../configuration/#overrides" >}}).

```
GET /distributor/ring
```

Displays a web page with the distributor hash ring status, including the state, healthy and last heartbeat time of each
distributor.

_For more information, check the page on [consistent hash ring]({{< relref "../operations/consistent_hash_ring" >}})._

### Ingesters ring status

```
GET /ingester/ring
```

Displays a web page with the ingesters hash ring status, including the state, healthy and last heartbeat time of each ingester.

_For more information, check the page on [consistent hash ring]({{< relref "../operations/consistent_hash_ring" >}})_

### Metrics-generator ring status

```
GET /metrics-generator/ring
```

Displays a web page with the metrics-generator hash ring status, including the state, health, and last heartbeat time of each metrics-generator.

This endpoint is only available when the metrics-generator is enabled. See [metrics-generator]({{< relref "../configuration/#metrics-generator" >}}).

_For more information, check the page on [consistent hash ring]({{< relref "../operations/consistent_hash_ring" >}})_

### Compactor ring status

```
GET /compactor/ring
```

Displays a web page with the compactor hash ring status, including the state, healthy and last heartbeat time of each
compactor.

_For more information, check the page on [consistent hash ring]({{< relref "../operations/consistent_hash_ring" >}})_

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

Displays a list of services and their status. If a service failed it will show the failure case.

```
GET /status/endpoints
```

Displays status information about the API endpoints.

```
GET /status/config
```

Displays the configuration.

Displays the configuration currently applied to Tempo (in YAML format), including default values and settings via CLI flags.
Sensitive data is masked. Please be aware that the exported configuration **doesn't include the per-tenant overrides**.

Optional query parameter:
- `mode = (diff|defaults)`: `diff` shows the difference between the default values and the current configuration. `defaults` shows the default values.

```
GET /status/runtime_config
```

Displays the override configuration.

Query parameter:
- `mode = (diff)`: Show the difference between defaults and overrides.

```
GET /status/usage-stats
```

Displays anonymous usage stats data that is reported back to Grafana Labs.
