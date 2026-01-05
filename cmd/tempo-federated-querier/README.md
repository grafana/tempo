# Tempo Federated Querier

The Tempo Federated Querier enables querying multiple Tempo instances simultaneously and combines the results into a unified view. This is useful when you have distributed tracing data split across multiple Tempo deployments (e.g., per region, per environment, or per team) and want to see a complete trace spanning all systems.

## Features

- Query multiple Tempo instances in parallel
- Combine and deduplicate trace spans from different sources
- Partial response support when some instances fail
- Metrics about which instances responded

## Use cases

- **Multi-region deployments**: Query traces across multiple geographic regions
- **Multi-tenant systems**: Combine traces from separate tenant-specific Tempo instances
- **Hybrid environments**: Query both on-premise and cloud Tempo deployments
- **Migration scenarios**: Query both old and new Tempo instances during migration

## Configuration

Create a configuration file (e.g., `config.yaml`):

```yaml
# Server configuration
http_listen_address: "0.0.0.0"
http_listen_port: 3200

# Tempo instances to federate
instances:
  - name: "tempo-region-us"
    endpoint: "http://tempo-us.example.com:3200"
    org_id: "my-tenant"
    timeout: 30s
  - name: "tempo-region-eu"
    endpoint: "http://tempo-eu.example.com:3200"
    org_id: "my-tenant"
    timeout: 30s
  - name: "tempo-region-asia"
    endpoint: "http://tempo-asia.example.com:3200"
    org_id: "my-tenant"
    timeout: 30s
  - name: "tempo-region-au"
    endpoint: "http://tempo-au.example.com:3200"
    org_id: "my-tenant"
    timeout: 30s

# Query settings
query_timeout: 30s
max_concurrent_queries: 20
max_bytes_per_trace: 52428800  # 50MB
allow_partial_responses: true
```

### Configuration options

| Option | Summary | Required | Type | Default |
| ------ | ------- | -------- | ---- | ------- |
| `http_listen_address` | Address for the HTTP server to listen on | No | String | `0.0.0.0` |
| `http_listen_port` | Port for the HTTP server to listen on | No | Integer | `3200` |
| `instances` | List of Tempo instances to query | Yes | List | N/A |
| `instances[].name` | Friendly name for the instance | No | String | Uses endpoint |
| `instances[].endpoint` | Base URL for the Tempo instance | Yes | String | N/A |
| `instances[].org_id` | Tenant ID (X-Scope-OrgID header) | No | String | Empty |
| `instances[].timeout` | Per-instance request timeout | No | Duration | Uses `query_timeout` |
| `instances[].headers` | Additional HTTP headers | No | Map | Empty |
| `query_timeout` | Default timeout for trace queries | No | Duration | `30s` |
| `max_concurrent_queries` | Maximum concurrent queries per request | No | Integer | `20` |
| `max_bytes_per_trace` | Maximum trace size in bytes | No | Integer | `52428800` (50MB) |
| `allow_partial_responses` | Return partial results on failures | No | Boolean | `true` |

## Usage

### Running locally

```bash
# Build the binary
go build -o tempo-federated-querier ./cmd/tempo-federated-querier

# Run with configuration
./tempo-federated-querier -config.file=config.yaml

# Print example configuration
./tempo-federated-querier -config.example

# Print version
./tempo-federated-querier -version
```

### Running with Docker

```bash
# Build the image
docker build -t tempo-federated-querier -f cmd/tempo-federated-querier/Dockerfile .

# Run with mounted config
docker run -p 3200:3200 -v $(pwd)/config.yaml:/config.yaml tempo-federated-querier -config.file=/config.yaml
```

### Running with Docker Compose

```yaml
version: '3.8'

services:
  tempo-federated-querier:
    build:
      context: .
      dockerfile: cmd/tempo-federated-querier/Dockerfile
    ports:
      - "3200:3200"
    volumes:
      - ./config.yaml:/config.yaml
    command: ["-config.file=/config.yaml"]
```

## API endpoints

The federated querier exposes a subset of Tempo's API for trace queries, search, and tag exploration.

### Trace by ID

```
GET /api/traces/{traceID}
GET /api/v2/traces/{traceID}
```

Queries all configured Tempo instances for the specified trace ID and combines the results. The v2 endpoint includes additional metadata about which instances responded.

### Search

```
GET /api/search?q={TraceQL}&limit={limit}&start={start}&end={end}
```

Searches all configured Tempo instances and combines the results. Traces are deduplicated by trace ID across instances, and metrics are aggregated.

**Query parameters:**

| Parameter | Description | Example |
| --------- | ----------- | ------- |
| `q` | TraceQL query | `{resource.service.name="my-service"}` |
| `limit` | Maximum traces to return | `20` |
| `start` | Start time (Unix seconds) | `1704067200` |
| `end` | End time (Unix seconds) | `1704153600` |
| `spss` | Spans per span set | `3` |

**Example:**

```bash
curl "http://localhost:3200/api/search?q=%7Bresource.service.name%3D%22order-service%22%7D&limit=20"
```

### Tags

```
GET /api/search/tags
GET /api/v2/search/tags
```

Returns all available tag names across all Tempo instances. Tags are deduplicated and sorted alphabetically. The v2 endpoint returns tags organized by scope (resource, span, etc.).

### Tag values

```
GET /api/search/tag/{tagName}/values
GET /api/v2/search/tag/{tagName}/values
```

Returns all values for a specific tag across all Tempo instances. Values are deduplicated and sorted. The v2 endpoint includes type information for each value.

**Example:**

```bash
# Get all values for service.name tag
curl "http://localhost:3200/api/search/tag/service.name/values"

# Response
{
  "tagValues": ["order-service", "payment-service", "user-service"]
}
```

### Status endpoints

```
GET /ready                    # Readiness check
GET /api/echo                 # Echo endpoint
GET /api/status/buildinfo     # Build information
GET /api/status/instances     # List of configured instances
```

## Grafana integration

Configure Grafana to use the federated querier as a Tempo data source:

1. Go to **Connections** > **Data sources**
1. Click **Add data source**
1. Select **Tempo**
1. Set the URL to your federated querier (e.g., `http://tempo-federated-querier:3200`)
1. Configure authentication if needed
1. Click **Save & test**

## Response metadata

The v2 trace endpoint returns additional metadata about the federated query:

```json
{
  "trace": { ... },
  "metrics": {
    "instancesQueried": 4,
    "instancesResponded": 4,
    "instancesFailed": 0,
    "totalSpans": 150,
    "partialResponse": false
  }
}
```

This helps you understand whether the trace data is complete or if some instances failed to respond.

## How it works

1. When a trace query arrives, the federated querier sends parallel requests to all configured Tempo instances
1. Responses are collected and combined
1. Spans are deduplicated using a hash of span ID and kind to avoid duplicates
1. The combined trace is sorted by start time and returned
1. If some instances fail but `allow_partial_responses` is true, partial results are returned with metadata indicating the failure

## Trace combining logic

The federated querier uses Tempo's native `trace.Combiner` from `pkg/model/trace` to combine spans from multiple instances. This ensures consistent deduplication and ordering with Tempo's internal logic.

### Architecture

The combiner package (`cmd/tempo-federated-querier/combiner`) handles all result combination:

```
combiner/
├── types.go      # QueryResult, CombineMetadata, SearchMetadata types
├── trace.go      # CombineTraceResults, CombineTraceResultsV2
├── search.go     # CombineSearchResults
└── tags.go       # CombineTagsResults, CombineTagValuesResults, etc.
```

### Deduplication

Spans are deduplicated based on their unique identity:

- **Span ID**: The unique identifier for each span
- **Span kind**: Client, server, producer, consumer, or internal

Two spans with the same span ID and kind from different instances are considered duplicates. The combiner keeps only one copy, preserving the most complete span data.

### Search result combining

When combining search results from multiple instances:

1. **Trace deduplication**: Traces with the same trace ID are merged
2. **Metadata merging**: For duplicate traces:
   - Earliest `startTimeUnixNano` is used
   - Longest `durationMs` is used
   - Service stats are combined using max values
   - Spansets are merged (deduplicated by span ID)
3. **Metrics aggregation**: All search metrics are summed:
   - `inspectedTraces`, `inspectedBytes`, `inspectedSpans`
   - `totalBlocks`, `completedJobs`, `totalJobs`
4. **Sorting**: Results are sorted by start time (most recent first)

### Tag combining

When combining tags and tag values:

1. **Tags**: All tag names are collected into a set, deduplicated, and sorted alphabetically
2. **Tag values**: All values are collected, deduplicated, and sorted
3. **Scopes (v2)**: Tags are grouped by scope (resource, span, etc.) with deduplication per scope
4. **Metrics**: Metadata metrics are summed across instances

### API response formats

The federated querier supports both JSON and Protocol Buffers (protobuf) response formats, matching Tempo's API:

| Accept Header | Response Format |
| ------------- | --------------- |
| `application/json` | JSON with OTLP-compatible structure |
| `application/protobuf` | Binary protobuf (tempopb.TraceByIDResponse) |

Grafana's Tempo data source uses protobuf by default for better performance.

### v1 vs v2 API differences

| Feature | v1 (`/api/traces/{traceID}`) | v2 (`/api/v2/traces/{traceID}`) |
| ------- | ---------------------------- | ------------------------------- |
| Response format | `tempopb.TraceByIDResponse` | `tempopb.TraceByIDResponse` |
| 404 on not found | Yes | No (returns empty trace) |
| Partial status | No | Yes (`status` field) |
| Upstream parsing | Direct trace JSON | Wrapped `{"trace": {...}}` |

### Handling 404 responses

When querying for a trace, individual Tempo instances may return 404 if they don't have that trace. The federated querier handles this gracefully:

- 404 responses are counted as "not found" but don't fail the query
- The trace is combined from instances that have it
- A 404 is only returned if **no** instance has the trace

This enables traces that span multiple systems to be viewed even when each system only has part of the trace.

### Error handling

The combiner tracks metadata about each instance:

| Metric | Description |
| ------ | ----------- |
| `instancesQueried` | Total number of instances queried |
| `instancesResponded` | Instances that returned a response (including 404) |
| `instancesWithTrace` | Instances that had the trace |
| `instancesNotFound` | Instances that returned 404 or empty trace |
| `instancesFailed` | Instances that returned an error |
| `totalSpans` | Total spans in the combined trace |
| `partialResponse` | True if any instance failed |

### Size limits

The `max_bytes_per_trace` configuration limits the combined trace size. When this limit is reached:

1. No more spans are added to the trace
1. The combiner returns what it has collected so far
1. This prevents memory issues with very large traces

## Limitations

- No TraceQL metrics queries (`/api/metrics/query`)
- No caching layer (relies on individual Tempo instance caching)
- Search results are limited by individual instance limits before combining

## Clock synchronization considerations

When combining traces from multiple Tempo instances, clock synchronization between the services that generate spans becomes critical. The federated querier combines spans based on their timestamps, and clock drift can cause issues.

### Common problems

| Problem | Cause | Effect |
| ------- | ----- | ------ |
| Parent appears after child | Clock on child service is ahead | Trace visualization shows impossible ordering |
| Gaps in trace timeline | Clock drift between services | Spans appear disconnected |
| Overlapping spans | Inconsistent clocks | Duration calculations become meaningless |
| Missing spans in time range queries | Clock offset exceeds query window | Spans fall outside the search time range |

### Example scenario

Consider a request flowing through services across different Tempo instances:

```
Service A (Instance 1) → Service B (Instance 2) → Service C (Instance 3)
   10:00:00.000            10:00:00.050            09:59:59.900
```

If Service C's clock is 150ms behind, its span appears to start *before* the request even reached Service A. The trace visualization becomes confusing and metrics like total duration are incorrect.

### What the federated querier does NOT do (currently)

The federated querier combines traces as-is without timestamp correction:

- **No clock skew detection**: Doesn't identify or warn about clock drift
- **No timestamp adjustment**: Doesn't modify span timestamps
- **No reordering based on causality**: Trusts the timestamps provided

This is by design—modifying timestamps could hide real latency issues or introduce other problems. The source of truth for timestamps should be fixed at the instrumentation level.

## Next steps

- Learn more about [Grafana Tempo](https://grafana.com/docs/tempo/latest/)
- Configure [distributed tracing](https://grafana.com/docs/tempo/latest/getting-started/) in your applications
- Set up [Grafana](https://grafana.com/docs/grafana/latest/) to visualize your traces
