---
title: Monitor query I/O and span timestamp distance
menuTitle: Query I/O and timestamp distance
description: Monitor query I/O and span timestamp quality with key Tempo metrics.
weight: 50
---

<!-- markdownlint-disable MD025 -->

# Monitor query I/O and span timestamp distance

You can use these metrics to monitor query I/O and span timestamp quality:

- `tempo_query_frontend_bytes_inspected_total` measures how many bytes a query inspects per request type and tenant. This value includes TraceQL metrics queries.
- `tempo_live_store_query_inspected_bytes_total` measures how many bytes live-store queries inspect per operation and tenant.
- `tempo_spans_distance_in_future_seconds` and `tempo_spans_distance_in_past_seconds` measure how far a span end time is from the ingestion time. This capability lets you find customers that send spans too far in the future or past, which may not be found using the Search API.

Use these metrics together to correlate query cost with data quality and pipeline health.

## Reference

The query frontend emits `tempo_query_frontend_bytes_inspected_total` when a request finishes, aggregating bytes inspected by the query path.
The `op` label uses `traces`, `search`, `metadata`, or `metrics`.

The live-store emits `tempo_live_store_query_inspected_bytes_total` while serving recent data.
Use it to separate recent-data query cost from the total cost reported by the query frontend.

The distributor emits `tempo_spans_distance_in_future_seconds` and `tempo_spans_distance_in_past_seconds` by comparing span end time with ingestion time.

| Names | Type | Labels | Buckets | Emitted | Notes |
|---|---|---|---|---|---|
| `tempo_query_frontend_bytes_inspected_total` | Counter | `tenant`, `op` | - | On request completion at the query frontend; aggregates bytes inspected by the query path; excludes cached querier responses. | Includes TraceQL metrics queries with `op="metrics"`. |
| `tempo_live_store_query_inspected_bytes_total` | Counter | `tenant`, `op` | - | During live-store query execution. | Live-store `op` values are `search`, `search_tags`, `search_tag_values`, `trace_by_id`, and `query_range`. |
| `tempo_spans_distance_in_future_seconds`, `tempo_spans_distance_in_past_seconds` | Histogram | `tenant` | 300s, 1800s, 3600s (5m, 30m, 1h) | In the distributor on ingest; observes seconds between span end time and ingestion time. | Spans in the future are accepted but invalid and might not be searchable. |

## PromQL examples

To see how frequently future-dated spans arrive by tenant, use the histogram count rate:

```promql
sum by (tenant) (
  rate(tempo_spans_distance_in_future_seconds_count[5m])
)
```

Inspect query read throughput (`bytes/s`) by tenant and operation:

```promql
sum by (tenant, op) (
  rate(tempo_query_frontend_bytes_inspected_total[5m])
)
```

Inspect recent-data query throughput from live-stores:

```promql
sum by (tenant, op) (
  rate(tempo_live_store_query_inspected_bytes_total[5m])
)
```

Top five tenants by inspected GiB over the last hour:

```promql
topk(
  5,
  sum by (tenant) (increase(tempo_query_frontend_bytes_inspected_total[1h])) / 1024 / 1024 / 1024
)
```

To quantify ingestion delay using the past-distance histogram, chart the P90 over time:

```promql
histogram_quantile(
  0.9,
  sum by (tenant, le) (
    rate(tempo_spans_distance_in_past_seconds_bucket[15m])
  )
)
```
