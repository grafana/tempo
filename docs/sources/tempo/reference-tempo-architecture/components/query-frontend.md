---
title: Query frontend
menuTitle: Query frontend
description: How the query frontend shards and distributes queries.
weight: 500
topicType: concept
versionDate: 2026-03-20
---

# Query frontend

The query frontend is the entry point for all queries in Tempo.
It receives TraceQL queries and trace ID lookups, shards them into parallel jobs,
and distributes those jobs to queriers for execution.

## How it works

The query frontend handles the full lifecycle of a query.
It shards a single query into many smaller jobs, each covering a subset of the data (for example, a subset of blocks or a time range).
Jobs are placed in a per-tenant queue and dispatched to queriers in batches, reducing round-trip overhead.
As queriers return partial results, the frontend merges and deduplicates them into a final response.
If a querier fails to process a job, the frontend retries it on another querier.
For search queries with a result limit, the frontend cancels remaining jobs as soon as enough results are collected.

## Job sharding

The frontend uses `target_bytes_per_job` to estimate how large each job should be.
Smaller values create more, smaller jobs (higher parallelism but more overhead).
Larger values create fewer, bigger jobs (less overhead but lower parallelism).

The total number of jobs for a query depends on the time range,
the volume of data in that range, and the `target_bytes_per_job` setting.

### Concurrent jobs

The `concurrent_jobs` setting controls how many jobs for a single query are dispatched to the queue at once.
If a query produces 5,000 jobs and `concurrent_jobs` is 1,000, only 1,000 jobs are active at a time.
As jobs complete, new ones are dispatched.

This limits the blast radius of a single large query.
In shared clusters, keeping this value lower ensures fair scheduling across tenants.

## Query range limits

A fixed `max_duration` is a poor proxy for query cost.
The same range can hold gigabytes for one tenant and terabytes for another.

`max_scanned_bytes`, under `query_frontend.search` or `query_frontend.metrics`, limits the range by the bytes of the backend blocks it scans.
The allowed range is the longest window ending at the query end whose backend blocks fit in `max_scanned_bytes`.
It's then raised to `min_duration` and capped at `max_duration`.
Longer queries are rejected with a 400 error that reports the allowed range and asks the user to reduce the time range. If the newest data in the range is already over the limit, the error says so instead.
A query of `min_duration` or less is always allowed.
If `min_duration` is above `max_duration`, `max_duration` wins.
Data newer than `query_backend_after` isn't counted.
`0` (default) disables the limit, and `min_duration` has no effect without `max_scanned_bytes`.

For example, this configuration lets a search span up to 30 days when its blocks fit in 500 GB,
and always allows 7 days:

```yaml
query_frontend:
  search:
    max_duration: 720h
    max_scanned_bytes: 500000000000  # 500 GB
    min_duration: 168h
  metrics:
    max_duration: 168h
    max_scanned_bytes: 100000000000  # 100 GB
    min_duration: 24h
```

Per-tenant overrides replace these values for one tenant.
They are `max_search_scanned_bytes`, `min_search_duration`, `max_metrics_scanned_bytes`, and `min_metrics_duration`,
set under `read` in the per-tenant overrides file.
An override of `0` uses the `query_frontend` value.
Each tenant of a multi-tenant metrics query is checked against its own limits.

```yaml
overrides:
  "<tenant-id>":
    read:
      max_search_scanned_bytes: 2000000000000  # 2 TB
      min_search_duration: 72h
      max_metrics_scanned_bytes: 200000000000  # 200 GB
      min_metrics_duration: 48h
```

To calibrate, the frontend logs a `msg="scanned bytes limit"` line for each tenant a query checks, including queries that `max_duration` rejects.
Each line has `estimated_scanned_bytes`, the estimated bytes the requested range scans, next to `max_scanned_bytes`, so the gap shows how far to raise the limit for that query.
Setting `min_duration` to each tenant's current `max_duration`, including per-tenant overrides, keeps every range allowed today.
This LogQL query finds queries that a `max_duration` of 7 days would newly allow:

```logql
{namespace="tempo", container="query-frontend"} |= "scanned bytes limit" | logfmt | result="rejected_max_duration" | passes_scanned_bytes="true" | requested_seconds <= 604800
```

## Querier connections

Queriers connect to the query frontend over streaming gRPC.
Each connection processes one batch at a time synchronously.
The number of concurrent connections from a querier determines how many batches it can process in parallel.

This is controlled by either `querier.max_concurrent_queries` (maximum total concurrent jobs per querier) or `querier.frontend_worker.parallelism` (number of connections per query frontend).

## Key configuration

```yaml
query_frontend:
  max_outstanding_per_tenant: 2000  # Max jobs in queue per tenant
  max_batch_size: 7                 # Jobs per batch sent to querier
  max_retries: 2                    # Retry count for failed jobs
  search:
    concurrent_jobs: 2000           # Max concurrent jobs per query
    target_bytes_per_job: 104857600 # ~100MB per job
```

Refer to [Tune search performance](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/backend_search/) for detailed tuning guidance.

## Key metrics

| Metric | Description |
|---|---|
| `tempo_query_frontend_queries_total` | Total queries received |
| `tempo_query_frontend_queue_length` | Current queue depth per tenant |

## Related resources

Refer to the [query-frontend configuration](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#query-frontend) for the full list of options.
