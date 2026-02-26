---
title: Manage trace ingestion
menuTitle: Manage trace ingestion
description: Size ingestion limits, fix refused spans, and identify what is driving trace volume in Tempo.
weight: 250
---

# Manage trace ingestion

In Tempo, distributors validate incoming spans against ingestion limits before writing them to Kafka.
If limits are too low for your workload, spans are refused and data is lost.
If limits are unchecked, ingestion volume can grow beyond what you intend to pay for.

This page helps you with three tasks:

- [Size ingestion limits](#size-ingestion-limits-for-your-workload) when deploying or reviewing your configuration.
- [Find and fix refused spans](#find-and-fix-refused-spans) when you are actively losing data.
- [Identify what is driving ingestion volume](#identify-what-is-driving-ingestion-volume) when costs are growing.

For an overview of how trace data flows through the write path, refer to [Tempo architecture](https://grafana.com/docs/tempo/<TEMPO_VERSION>/introduction/architecture/).

## Size ingestion limits for your workload

Tempo enforces three ingestion limits. Understanding what the defaults mean for your traffic helps you set appropriate values at deploy time, rather than discovering them through production incidents.

### Rate limit

`rate_limit_bytes` (default: 15,000,000) sets the sustained byte rate each distributor allows per tenant, measured in bytes per second.
For a typical span size of around 500 bytes, the default accommodates roughly 30,000 spans per second per distributor.

How this scales depends on your `rate_strategy`:

- **`local`** (default): each distributor enforces the limit independently. With three distributors, the effective cluster limit is approximately 90,000 spans per second.
- **`global`**: the configured rate is shared across all distributors. The total cluster rate equals the configured value regardless of how many distributors you run.

`burst_size_bytes` (default: 20,000,000) allows temporary spikes above the sustained rate, for example during application deployments. The burst allowance is always applied locally, regardless of rate strategy.

### Live trace limit

`max_traces_per_user` (default: 10,000) caps the number of concurrently active traces per tenant on each live-store.
If your services produce many short-lived traces in parallel, you may need to raise this.

`max_global_traces_per_user` (default: 0, disabled) sets a cluster-wide cap instead of a per-instance cap.

### Per-trace size limit

`max_bytes_per_trace` (default: 5,000,000) caps the total size of a single trace.
Traces that exceed this limit are partially dropped.
Unusually large traces often indicate a retry loop or misconfigured instrumentation rather than normal application behavior.

### Example configuration

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

## Find and fix refused spans

When the distributor refuses spans, it logs an error and increments the `tempo_discarded_spans_total` metric.
The error message tells you which limit was exceeded.

### Error reference

The following table lists the three error types, what each one means, and how to fix it.

| Error | Cause | Fix |
|---|---|---|
| `RATE_LIMITED` | The tenant's byte rate exceeded `rate_limit_bytes`. | Raise `rate_limit_bytes`, or add distributors if using `rate_strategy: local`. If volume is genuinely higher than intended, reduce it upstream with [sampling](https://grafana.com/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/instrument-send/set-up-collector/tail-sampling/). |
| `LIVE_TRACES_EXCEEDED` | The number of concurrent active traces on a live-store exceeded `max_traces_per_user`. | Raise `max_traces_per_user`, or set `max_global_traces_per_user` to distribute the limit across the cluster. |
| `TRACE_TOO_LARGE` | A single trace exceeded `max_bytes_per_trace` (default 5 MB). | Raise `max_bytes_per_trace` in the `global` overrides. Also investigate why the trace is so large. Common causes include retry loops and misconfigured instrumentation. |

### Check which limit is being hit

Query the `tempo_discarded_spans_total` metric.
The `reason` label indicates which limit caused the refusal:

```promql
sum by (reason) (rate(tempo_discarded_spans_total[5m]))
```

### Log refused spans for debugging

To log individual refused spans with their trace IDs, enable `log_discarded_spans` in the distributor configuration:

```yaml
distributor:
  log_discarded_spans:
    enabled: true
```

Set `include_all_attributes: true` for more verbose output that includes span attributes.
Refer to [Distributor refusing spans](https://grafana.com/docs/tempo/<TEMPO_VERSION>/troubleshooting/send-traces/max-trace-limit-reached/) for additional troubleshooting steps.

### Traces missing from queries without errors

If the distributor is not refusing spans but traces are missing from query results, the issue may be downstream.
Live-stores consume trace data from Kafka and serve recent queries.
If a live-store falls behind its Kafka partition, query results may be incomplete.

The `fail_on_high_lag` setting (default `false`) controls this behavior:

- When `false`, the live-store returns whatever data it has, which may be incomplete.
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

Use the following query to find which services are sending the most data:

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
