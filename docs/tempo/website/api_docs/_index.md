---
title: API documentation
weight: 350
---

# Tempo's API

Tempo exposes an API for pushing and querying traces, and operating the cluster itself.

For the sake of clarity, in this document we have grouped API endpoints by service, but keep in mind that they're exposed both when running Tempo in microservices and singly-binary mode:
- **Microservices**: each service exposes its own endpoints
- **Single-binary**: the Tempo process exposes all API endpoints for the services running internally

## Endpoints

| API | Service | Type | Endpoint |
| --- | ------- | ---- | -------- |
| [Readiness probe](#readiness-probe) | _All services_ |  HTTP | `GET /ready` |
| [Metrics](#metrics) | _All services_ |  HTTP | `GET /metrics` |
| [Pprof](#pprof) | _All services_ |  HTTP | `GET /debug/pprof` |
| [Ingest traces](#ingest) | Distributor |  - | See section for details |
| [Querying traces](#query) | Query-frontend |  HTTP | `GET /api/traces/<traceID>` |
| [Searching traces](#search) | Query-frontend | HTTP | `GET /api/search?<params>` |
| [Search tag names](#search-tags) | Query-frontend | HTTP | `GET /api/search/tags` |
| [Search tag values](#search-tag-values) | Query-frontend | HTTP | `GET /api/search/tag/<tag>/values` |
| [Query Echo Endpoint](#query-echo-endpoint) | Query-frontend |  HTTP | `GET /api/echo` |
| [Memberlist](#memberlist) | Distributor, Ingester, Querier, Compactor |  HTTP | `GET /memberlist` |
| [Flush](#flush) | Ingester |  HTTP | `GET,POST /flush` |
| [Shutdown](#shutdown) | Ingester |  HTTP | `GET,POST /shutdown` |
| [Distributor ring status](#distributor-ring-status) (*) | Distributor |  HTTP | `GET /distributor/ring` |
| [Ingesters ring status](#ingesters-ring-status) | Distributor, Querier |  HTTP | `GET /ingester/ring` |
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

Tempo distributor uses the OpenTelemetry Receivers as a shim to ingest trace data.
Note that these APIs are meant to be consumed by the corresponding client SDK or a pipeline service like Grafana
Agent / OpenTelemetry Collector / Jaeger Agent.

|  Protocol | Type | Docs |
|  -------- | ---- | ---- |
|  OpenTelemetry | GRPC | [Link](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/protocol/otlp.md) |
|  OpenTelemetry | HTTP | [Link](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/protocol/otlp.md) |
|  Jaeger | Thrift Compact | [Link](https://www.jaegertracing.io/docs/latest/apis/#span-reporting-apis) |
|  Jaeger | Thrift Binary | [Link](https://www.jaegertracing.io/docs/latest/apis/#span-reporting-apis) |
|  Jaeger | Thrift HTTP |  [Link](https://www.jaegertracing.io/docs/latest/apis/#span-reporting-apis) |
|  Jaeger | GRPC | [Link](https://www.jaegertracing.io/docs/latest/apis/#span-reporting-apis) |
|  Zipkin | HTTP | [Link](https://zipkin.io/zipkin-api/) |

_For information on how to use the Zipkin endpoint with curl (for debugging purposes) check [here](pushing-spans-with-http)._

### Query

Tempo's Query API is simple. The following request is used to retrieve a trace from the query frontend service in
a microservices deployment, or the Tempo endpoint in a single binary deployment.

```
GET /api/traces/<traceid>
```

The following query API is also provided on the querier service for _debugging_ purposes.

```
GET /querier/api/traces/<traceid>?mode=xxxx&blockStart=0000&blockEnd=FFFF
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

Note that this API is not meant to be used directly unless for debugging the sharding functionality of the query
frontend.

Returns:
By default this endpoint returns [OpenTelemetry](https://github.com/open-telemetry/opentelemetry-proto/tree/main/opentelemetry/proto/trace/v1) JSON,
but if it can also send OpenTelemetry proto if `Accept: application/protobuf` is passed.

### Search

<span style="background-color:#f3f973;">This experimental endpoint is disabled by default and can be enabled via the `search_enabled` YAML config option.</span>

Tempo's Search API finds traces based on span and process attributes (tags and values).  The API is available in the query frontend service in
a microservices deployment, or the Tempo endpoint in a single binary deployment.  The following request is used to find traces containing spans
from service "myservice" and the url contains "api/myapi".

```
GET /api/search?tags=service.name%3Dmyservice%20http.url%3Dapi%2Fmyapi
```

The URL query parameters support the following values:
- `tags = (logfmt)`: logfmt encoding of any span-level or process-level attributes to filter on. The value is matched as a case-insensitive substring. Key-value pairs are separated by spaces. If a value contains a space, it should be enclosed within double quotes.
- `minDuration = (go duration value)`
  Optional.  Find traces with at least this duration.  Duration values are of the form `10s` for 10 seconds, `100ms`, `30m`, etc.
- `maxDuration = (go duration value)`
  Optional.  Find traces with no greater than this duration.  Uses the same form as `minDuration`.
- `limit = (integer)`
  Optional.  Limit the number of search results. Default is 20, but this is configurable in the querier. Refer to [Configuration](../configuration#querier).

#### Example

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

### Search Tags

<span style="background-color:#f3f973;">This experimental endpoint is disabled by default and can be enabled via the `search_enabled` YAML config option.</span>

This endpoint retrieves all discovered tag names that can be used in search.  The endpoint is available in the query frontend service in
a microservices deployment, or the Tempo endpoint in a single binary deployment.

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

### Search Tag Values

<span style="background-color:#f3f973;">This experimental endpoint is disabled by default and can be enabled via the `search_enabled` YAML config option.</span>

This endpoint retrieves all discovered values for the given tag, which can be used in search.  The endpoint is available in the query frontend service in
a microservices deployment, or the Tempo endpoint in a single binary deployment.  The following request will return all discovered service names.

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

> Note: this endpoint is only available when Tempo is configured with [the global override strategy](../configuration#ingestion-limits).

```
GET /distributor/ring
```

Displays a web page with the distributor hash ring status, including the state, healthy and last heartbeat time of each
distributor.

_For more information, check the page on [consistent hash ring](../operations/consistent_hash_ring)._

### Ingesters ring status

```
GET /ingester/ring
```

Displays a web page with the ingesters hash ring status, including the state, healthy and last heartbeat time of each ingester.

_For more information, check the page on [consistent hash ring](../operations/consistent_hash_ring)._



### Compactor ring status

```
GET /compactor/ring
```

Displays a web page with the compactor hash ring status, including the state, healthy and last heartbeat time of each
compactor.

_For more information, check the page on [consistent hash ring](../operations/consistent_hash_ring)._

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

Optional Query Parameter:
- `mode = (diff|defaults)`: Used to show the difference between the default values and the current configuration if `diff` is given. The default values will be shown if `defaults` is used.

```
GET /status/runtime_config
```

Displays the override configuration.

Query Parameter:
- `mode = (diff)`: Used to show the difference between defaults and overrides.

Displays the configuration currently applied to Tempo (in YAML format), including default values and settings via CLI flags.
Sensitive data is masked. Please be aware that the exported configuration **doesn't include the per-tenant overrides**.
