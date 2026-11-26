---
title: User-configurable overrides
menuTitle: Configure Tempo overrides through the user-configurable overrides API
description: Configure Tempo overrides through the user-configurable overrides API
weight: 300
aliases:
  - ../use-configurable-overrides/ # https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/user-configurable-overrides/
---

# User-configurable overrides

User-configurable overrides in Tempo let you change overrides for your tenant using an API.
Instead of modifying a file or Kubernetes `configmap`, you (and other services relying on Tempo) can use this API to
modify the overrides directly.

## Architecture

User-configurable overrides are stored in an object store bucket managed by Tempo.

![user-configurable-overrides-architecture.svg](../user-configurable-overrides-architecture.svg)

{{< admonition type="note" >}}
We recommend using a different bucket for overrides and traces storage, but they can share a bucket if needed.
When sharing a bucket, make sure any lifecycle rules are scoped correctly to not remove data of user-configurable
overrides module.
{{< /admonition >}}

Overrides of every tenant are stored at `/{tenant name}/overrides.json`:

```
overrides/
├── 1/
│   └── overrides.json
└── 2/
    └── overrides.json
```

Tempo regularly polls this bucket and keeps a copy of the limits in-memory. When requesting the overrides for a tenant,
the overrides module:

1. Checks this override is set in the user-configurable overrides, if so return that value.
2. Checks if this override is set in the runtime configuration (`configmap`), if so return that value.
3. Returns the default value.

### Supported fields

User-configurable overrides are designed to be a subset of the runtime overrides. Refer
to [Overrides](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#overrides) for information about all
overrides.
When you set a field in both the user-configurable overrides and the runtime overrides, the value from the
user-configurable overrides takes priority.

{{< admonition type="note" >}}
In Tempo 2.x, `metrics_generator.processors` user-configurable override field is OR-merged with the runtime overrides list.

In Tempo 3.0, if `processors` is set in user-configurable overrides, that will override the runtime `metrics_generator.processors` config. Setting `processors: []` in user-configurable overrides disables all processors for the tenant.
{{< /admonition >}}

{{< admonition type="warning" >}}
The `local-blocks` processor was removed in Tempo 3.0. TraceQL metrics queries on recent data are now served by the live-store instead. If your overrides reference `local-blocks`, remove it before upgrading.
For details, refer to [Metrics generator](https://grafana.com/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/metrics-generator/).
{{< /admonition >}}

```yaml
[forwarders: <list of strings>]

cost_attribution:
  [dimensions: <map string to string>]

metrics_generator:

  [processors: <list of strings>]
  [collection_interval: <duration>]
  [trace_id_label_name: <string>]
  [ingestion_time_range_slack: <duration>]
  [disable_collection: <bool> | default = false]
  [generate_native_histograms: <classic|native|both> | default = classic]
  [native_histogram_max_bucket_number: <int> | default = 100]
  [native_histogram_bucket_factor: <float> | default = 1.1]
  [native_histogram_min_reset_duration: <duration> | default = 15m]
  [span_name_sanitization: <string>]

  processor:

    service_graphs:
      [enable_service_graph_edge_storage: <bool>]
      [bypass_services: <list of string>]
      [bypass_service_regex: <list of string>]
      [histogram_buckets: <list of float>]
      [dimensions: <list of string>]
      [peer_attributes: <list of string>]
      [enable_client_server_prefix: <bool>]
      [enable_messaging_system_latency_histogram: <bool>]
      [enable_virtual_node_label: <bool>]
      [span_multiplier_key: <string>]
      [enable_tracestate_span_multiplier: <bool>]
      [filter_policies: [
        [
          include/include_any/exclude:
            match_type: <string> # options: strict, regex
            attributes:
              - key: <string>
                value: <any>
        ]
      ]]
    span_metrics:
      [histogram_buckets: <list of float>]
      [dimensions: <list of string>]
      [dimension_mappings: <list of mappings {name: <string>, source_labels: <list of string>, join: <string>}>]
      [intrinsic_dimensions: <map string to bool>]
      [filter_policies: [
        [
          include/include_any/exclude:
            match_type: <string> # options: strict, regex
            attributes:
              - key: <string>
                value: <any>
        ]
      ]]
      [enable_target_info: <bool>]
      [target_info_excluded_dimensions: <list of string>]
      [enable_instance_label: <bool>]
      [span_multiplier_key: <string>]
      [enable_tracestate_span_multiplier: <bool>]

    host_info:
      [host_identifiers: <list of string>]
      [metric_name: <string>]
```

### Store full service graph relationships

Set `metrics_generator.processor.service_graphs.enable_service_graph_edge_storage` to `true`
to persist individual parent/child edge records with trace data
and generate service graph metrics from all span kinds.
The flag defaults to `false` and supports per-tenant overrides.
It replaces the development name `enable_span_kind_relaxation`.

```yaml
metrics_generator:
  processor:
    service_graphs:
      enable_service_graph_edge_storage: true
```

Edge records are written into completed `vParquet4` and `vParquet5` trace blocks,
using Tempo's configured trace storage backend and trace retention.
They do not depend on the generator's in-memory matching cache surviving.
The `wait` and `max_items` settings bound that cache;
they do not expire records already written to trace blocks.
Original spans remain unchanged.
All-span metrics settle as their endpoint `wait` windows expire.
Each span has its own wait budget, so earlier spans do not cut off newer relationships.
This allows shared client/server span IDs to arrive in either order
before relationships are counted.
Strict matching keeps its existing timing.

Trace-by-ID responses include the persisted records in `serviceGraphEdges`.
Records contain parent and child span IDs, service names, span kinds,
and a `parentMissing` marker for unresolved parent references.
Compaction and query combination can resolve references across trace fragments.
No collapsed or synthetic spans are stored.
Enabling the flag does not backfill previously written blocks.
Compaction preserves and reconciles existing edge records;
it never generates new records from historical spans or overrides a tenant opt-out.
Turn it on only after the components reading, writing, and compacting these blocks
have been upgraded to support the added records.

This uses more memory and disk space.
The block builder and live-store honor tenant-specific overrides;
turning the flag off stops creating new records and preserves existing records.
The trace backend must use `vParquet4` or `vParquet5` when enabling edge storage.

### Presentation blacklist

`bypass_services` contains exact service names,
and `bypass_service_regex` contains Go regular expressions.
These are presentation settings for a graph consumer.
Lists configured under `metrics_generator.processor.service_graphs` provide fallback defaults
when `overrides.defaults.metrics_generator.processor.service_graphs` leaves a list unset.
Tenant and request overrides replace each list independently;
an explicit empty list clears it.
Saved configuration and query parameters share validation limits:
128 total rules and 65,536 total UTF-8 bytes across both lists,
with nonempty strings and valid Go regular expressions.
If a later runtime reload conflicts with saved API lists,
queries retain the last valid presentation pair until a compatible update arrives.
The reload failure counter and logs report the incompatible combination;
other runtime overrides continue to reload.
They never drop spans, collapse persisted relationships,
or change the full edge set emitted as metrics.
Regex matching searches within a name;
use `^` and `$` to match the whole name.

For `A → gateway → proxy → B`,
Tempo retains the spans and the three original edge records.
A visualization can use the blacklist to hide the intermediate services
without losing the original relationships on disk.
This Tempo change does not implement a Grafana UI toggle.

### Optional API updates

API changes to edge-storage and presentation settings are disabled by default.
To allow them, enable the user-configurable overrides module
and its separate API opt-in:

```yaml
overrides:
  user_configurable_overrides:
    enabled: true
    api:
      enable_service_graph_bypass: true
```

The module also needs its storage client configured as described below.
That client saves tenant configuration;
edge records themselves use Tempo's trace storage backend.
Runtime YAML overrides do not require the API opt-in.

With `enable_service_graph_bypass: false`,
API requests that change the edge-storage flag or blacklist return `400`.
This includes deleting overrides containing those settings.
Reads and unrelated updates remain available,
including updates preserving previously saved settings.
The API flag controls writes rather than disabling saved settings.

Use `PATCH /api/overrides` to update only these fields
without replacing unrelated tenant settings:

```shell
curl -X PATCH "$TEMPO_URL/api/overrides" \
  -H "X-Scope-OrgID: $TENANT_ID" \
  -H 'Content-Type: application/json' \
  --data '{"metrics_generator":{"processor":{"service_graphs":{"enable_service_graph_edge_storage":true,"bypass_services":["gateway"],"bypass_service_regex":["^proxy-.*$"]}}}}'
```

Omitting a field retains its inherited value.
Setting a blacklist list to `[]` clears that list for the tenant.
Presentation settings can be cleared without disabling edge storage.
Saved settings become effective after overrides and component configuration reloads.
Changes to matching configuration recreate the generator processor and may reset metric series;
this does not delete records already stored in trace blocks.

### Filter policy validation

Filter policies submitted through this API are fully validated. Tempo checks that attribute keys are valid TraceQL identifiers, only `resource` and `span` scopes are supported, regular expression patterns compile, and intrinsic values (`kind`, `status`) use recognized values. Invalid policies are rejected with a descriptive error.

For the complete list of validation rules and supported values, refer to [Filtering](/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/span-metrics/span-metrics-metrics-generator/#filtering).

## API

All API requests are handled on the `/api/overrides` endpoint. The module supports `GET`, `POST`, `PATCH`, and `DELETE`
requests. If you set [`http_api_prefix`](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#server) in your Tempo configuration, prepend it to the path (for example, `/tempo/api/overrides`).

This endpoint is tenant-specific. If Tempo is run in multitenant mode, all requests should have an appropriate
`X-Scope-OrgID` header.

If the tenant is run in distributed mode, only the query-frontend will accept API requests.

### Operations

#### GET /api/overrides

Returns the current overrides and its version.

Query-parameters:

- `scope`: whether to return overrides from the API only `api` or merge it with the runtime overrides `merged`. Defaults
  to `api`.

Example:

```shell
curl -X GET -v -H "X-Scope-OrgID: 3" http://localhost:3100/api/overrides\?scope=merged
```

#### POST /api/overrides

Update the overrides with the given payload. Note this overwrites any existing overrides.

Example:

```shell
curl -X POST -v -H "X-Scope-OrgID: 3" -H "If-Match: 1697726795401423" http://localhost:3100/api/overrides --data "{}"
```

#### PATCH /api/overrides

Update the existing overrides by patching it with the payload.
It follows the JSON merge patch protocol ([RFC 7386](https://datatracker.ietf.org/doc/html/rfc7386)).

Example:

```shell
curl -X PATCH -v -H "X-Scope-OrgID: 3" http://localhost:3100/api/overrides --data "{\"forwarders\":null}"
```

#### DELETE /api/overrides

Delete the existing overrides.

Example:

```shell
curl -X DELETE -H "X-Scope-OrgID: 3" -H "If-Match: 1697726795401423" http://localhost:3100/api/overrides
```

### Versioning

To handle concurrent read and write operations, the backend stores the overrides with a version.
Whenever the overrides are returned, the response has an Etag header with the current version.

```shell
$ curl -v http://localhost:3100/api/overrides
...
< HTTP/1.1 200 OK
< Content-Type: application/json
< Etag: 1697726795401423
< Date: Wed, 07 Feb 2024 17:49:04 GMT
< Content-Length: 118
...
```

Requests that modify or delete overrides need to pass the current version using the `If-Match` header:

```shell
curl -X POST -H "If-Match: 1697726795401423" http://localhost:3100/api/overrides --data "..."
```

This example uses overrides in the `overrides.json` file with the location in `pwd`:

```shell
curl -X POST -H "X-Scope-OrgID: 3" -H "If-Match: 1697726795401423" http://localhost:3100/api/overrides --data @overrides.json
```

If the version doesn't match the version in the backend, the request is rejected with HTTP error 412.

### Conflicting runtime overrides check

Overrides set through the user-configurable overrides take priority over runtime overrides.
This can lead to misleading scenarios because a value set in the runtime overrides is not actively being used.

To warn users about preexisting runtime overrides, there is an optional check for conflicting runtime overrides.
If enabled requests are rejected if:

1. There are no user-configurable overrides yet for this tenant.
2. There are runtime overrides set that contain overrides present in the user-configurable overrides.

The check can be enabled in the configuration:

```yaml
overrides:
  user_configurable_overrides:
    api:
      check_for_conflicting_runtime_overrides: true
```

You can bypass this check by setting the query parameter `skip-conflicting-overrides-check=true`:

```shell
curl -X POST -H "If-Match: 1697726795401423" http://localhost:3100/api/overrides?skip-conflicting-overrides-check=true --data "..."
```
