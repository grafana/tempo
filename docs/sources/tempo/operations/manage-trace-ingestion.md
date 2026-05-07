---
title: Manage trace ingestion
menuTitle: Manage trace ingestion
description: Size ingestion limits, fix refused spans, and identify what is driving trace volume in Tempo.
weight: 250
---

# Manage trace ingestion

If you are seeing `RATE_LIMITED`, `LIVE_TRACES_EXCEEDED`, or `TRACE_TOO_LARGE` errors, or if your trace storage costs are rising unexpectedly, this page can help.

Grafana Tempo enforces ingestion limits at multiple points in the write path.
The distributor checks rate limits before writing spans to Kafka.
Downstream, live-stores enforce per-trace size and live trace count limits asynchronously, and block-builders enforce per-trace size limits.
If limits are too low for your workload, spans are discarded and data is lost.
If limits are unchecked, ingestion volume can grow beyond what you intended.

This page covers three tasks:

- [Size ingestion limits](#size-ingestion-limits-for-your-workload) when deploying or reviewing your configuration.
- [Find and fix discarded spans](#find-and-fix-discarded-spans) when you are actively losing data.
- [Identify what is driving ingestion volume](#identify-what-is-driving-ingestion-volume) when costs are growing.

For an overview of how trace data flows through the write path, refer to [Tempo architecture](https://grafana.com/docs/tempo/<TEMPO_VERSION>/introduction/architecture/).

## Size ingestion limits for your workload

Tempo enforces three ingestion limits. Understanding what the defaults mean for your traffic helps you set appropriate values at deploy time, rather than discovering them through production incidents.

### Rate limit

`rate_limit_bytes` (default: `15,000,000`) sets the sustained byte rate each distributor allows per tenant, measured in bytes per second.
For a typical span size of around 500 bytes, the default accommodates roughly 30,000 spans per second per distributor.

How this scales depends on your `rate_strategy`:

- **`local`** (default): each distributor enforces the limit independently. With three distributors, the effective cluster limit is approximately 90,000 spans per second.
- **`global`**: the configured rate is shared across all distributors. The total cluster rate equals the configured value regardless of how many distributors you run.

`burst_size_bytes` (default: `20,000,000`) allows temporary spikes above the sustained rate, for example during application deployments. The burst allowance is always applied locally, regardless of rate strategy.

### Live trace limit

`max_traces_per_user` (default: `10,000`) caps the number of concurrently active traces per tenant on each live-store.
This limit is enforced asynchronously in the live-store, not at ingestion time in the distributor.
Block-builders do not enforce this limit.
If your services produce many short-lived traces in parallel, you may need to raise this.

{{< admonition type="note" >}}
The `max_global_traces_per_user` setting, which provides a cluster-wide cap for the ingester write path, has been moved to `ingestion.max_global_traces_per_user` in Tempo 3.0.
{{< /admonition >}}

### Per-trace size limit

`max_bytes_per_trace` (default: `5,000,000`) caps the total size of a single trace.
This limit is enforced asynchronously in live-stores and block-builders.
Traces that exceed this limit are partially dropped.
Unusually large traces often indicate a retry loop or misconfigured instrumentation rather than normal application behavior.

### Example configuration

To estimate the rate limit you need, multiply your average span size by your peak spans-per-second across all services for a given tenant.

The following example raises the defaults for a high-throughput workload:

```yaml
overrides:
  defaults:
    ingestion:
      rate_strategy: local
      rate_limit_bytes: 30000000
      burst_size_bytes: 40000000
      max_traces_per_user: 50000
    global:
      max_bytes_per_trace: 10000000
```

If you run a multi-tenant deployment, you can set different limits per tenant using runtime overrides instead of raising the global defaults.
Refer to [Enable multi-tenancy](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/manage-advanced-systems/multitenancy/) for per-tenant override examples.

For the full list of available settings, refer to [Ingestion limits](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#ingestion-limits) in the configuration reference.
You can also manage per-tenant limits through the API using [user-configurable overrides](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/manage-advanced-systems/user-configurable-overrides/).

## Find and fix discarded spans

When a span exceeds an ingestion limit or fails validation, Tempo discards it and increments the `tempo_discarded_spans_total` metric.
The distributor rejects entire push requests that exceed the rate limit or contain invalid trace or span IDs, before any spans reach Kafka.
Live-stores discard spans that exceed per-trace size or live trace count limits after consuming them from Kafka.
Block-builders discard spans that exceed per-trace size limits.

### Error reference

The following table lists the three error types, what each one means, and how to fix it.

| Error                  | Cause                                                                                  | Fix                                                                                                                                                                                                                                                                                |
| ---------------------- | -------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `RATE_LIMITED`         | The tenant's byte rate exceeded `rate_limit_bytes`.                                    | Raise `rate_limit_bytes`, or add distributors if using `rate_strategy: local`. If volume is genuinely higher than intended, reduce it upstream with [sampling](https://grafana.com/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/instrument-send/set-up-collector/tail-sampling/). |
| `LIVE_TRACES_EXCEEDED` | The number of concurrent active traces on a live-store exceeded `max_traces_per_user`. | Raise `max_traces_per_user`, or add live-store instances to distribute the active trace count across more nodes.                                                                                                                                                                    |
| `TRACE_TOO_LARGE`      | A single trace exceeded `max_bytes_per_trace` (default 5 MB).                          | Raise `max_bytes_per_trace` in the `global` overrides. Also investigate why the trace is so large. Common causes include retry loops and misconfigured instrumentation.                                                                                                            |

### Check why spans are being discarded

Query the `tempo_discarded_spans_total` metric.
The `reason` label identifies why Tempo discarded each span:

```promql
sum by (reason) (rate(tempo_discarded_spans_total[5m]))
```

The following table lists the possible `reason` values:

| Reason                        | Meaning                                                     | Component              |
| ----------------------------- | ----------------------------------------------------------- | ---------------------- |
| `rate_limited`                | Tenant byte rate exceeded `rate_limit_bytes`.               | Distributor            |
| `trace_too_large`             | Single trace exceeded `max_bytes_per_trace`.                | Live-store, block-builder |
| `live_traces_exceeded`        | Active trace count exceeded `max_traces_per_user`.          | Live-store             |
| `invalid_trace_id`            | Batch contained a trace ID that isn't 128 bits.             | Distributor            |
| `invalid_span_id`             | Batch contained a span ID that isn't 64 bits or was all zeros. | Distributor         |
| `trace_too_large_to_compact`  | Trace too large for the backend-worker to compact.          | Backend-worker         |
| `unknown_error`               | Unexpected error during span processing.                    | Live-store             |

### Log discarded spans for debugging

To log spans discarded by the distributor (rate-limited spans) with their trace IDs, enable `log_discarded_spans` in the distributor configuration:

```yaml
distributor:
  log_discarded_spans:
    enabled: true
```

Set `include_all_attributes: true` for more verbose output that includes span attributes.

Spans discarded by live-stores for `LIVE_TRACES_EXCEEDED` or `TRACE_TOO_LARGE` are logged at debug level by the live-store.
To see these entries, set the live-store log level to `debug`.

Refer to [Distributor refusing spans](https://grafana.com/docs/tempo/<TEMPO_VERSION>/troubleshooting/send-traces/max-trace-limit-reached/) for additional troubleshooting steps.

### Traces missing from queries without errors

If the distributor is not refusing spans but traces are missing from query results, the issue may be downstream.
Live-stores consume trace data from Kafka and serve recent queries.
If a live-store falls behind its Kafka partition, query results may be incomplete.

Monitor the `tempo_live_store_lagged_requests_total` metric to detect when this happens.
This counter increments every time a search or metrics query hits a live-store whose Kafka lag overlaps the requested time range, meaning results may be incomplete.
The metric is labeled by `route` (`/tempopb.Querier/SearchRecent` or `/tempopb.Metrics/QueryRange`).

The `fail_on_high_lag` setting (default `false`) controls how the live-store responds when lag is detected:

- When `false`, the live-store returns whatever data it has, which may be incomplete. The metric still increments.
- When `true`, the live-store returns an error when it cannot guarantee completeness.

Refer to [Unable to find traces](https://grafana.com/docs/tempo/<TEMPO_VERSION>/troubleshooting/querying/unable-to-see-trace/) for query-side troubleshooting.

## Identify what is driving ingestion volume

When your cluster is healthy but ingestion is growing, the first step is finding which services are responsible for the most volume.

### Set up cost attribution

The usage tracker breaks down ingested bytes by configurable attributes, giving you a per-service view of who is consuming capacity.

Enable cost attribution in the distributor and configure which attributes to track in overrides:

```yaml
distributor:
  usage:
    cost_attribution:
      enabled: true

overrides:
  defaults:
    cost_attribution:
      dimensions:
        resource.service.name: "service"
```

### Find the top contributors

After enabling cost attribution, the distributor exposes the `tempo_usage_tracker_bytes_received_total` metric on the `/usage_metrics` endpoint, labeled by the dimensions you configured.

You can query this endpoint directly:

```bash
curl http://<distributor-host>:3200/usage_metrics
```

If you scrape this endpoint with Prometheus, you can use the following query to find which services are sending the most data:

```promql
topk(10,
  sum by (service) (
    rate(tempo_usage_tracker_bytes_received_total[1h])
  )
)
```

For the full set of configuration options, including scoping dimensions by resource or span and customizing label names, refer to [Usage tracker](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/usage-tracker/).

### Reduce volume from noisy services

After you know which services are driving volume, the most effective way to reduce it is [sampling](https://grafana.com/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/instrument-send/set-up-collector/tail-sampling/) at the collector layer.
Tail sampling lets you keep traces with errors or high latency while dropping routine ones, reducing volume without losing visibility into the problems that matter.
