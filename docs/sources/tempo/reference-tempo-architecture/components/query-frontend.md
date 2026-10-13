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

The frontend rejects TraceQL searches and metrics queries whose time range is too long.
`max_duration` sets a fixed ceiling for each query type.

You can also limit the range by the bytes it scans with `max_scanned_bytes`, under `query_frontend.search` or `query_frontend.metrics`.
The frontend sums the size of the backend blocks the query range overlaps.
When the sum exceeds `max_scanned_bytes`, the allowed range shrinks to the longest window ending at the query end that fits the budget.
`min_duration` sets a floor that is always allowed, and `max_duration` stays the ceiling.
Per-tenant overrides are `max_search_scanned_bytes`, `min_search_duration`, `max_metrics_scanned_bytes`, and `min_metrics_duration`.
For a multi-tenant metrics query, each tenant is checked against its own limits.
With `max_scanned_bytes` set, that includes each tenant's own `max_metrics_duration`.
Without it, only the `max_metrics_duration` of the combined tenant ID applies, as before.
Rejected queries get an error that reports the allowed range.
The `tempo_query_frontend_scanned_bytes_limit_derived_range_seconds` histogram records the range the budget allows, before the floor and ceiling,
for queries within `max_duration` that are over the budget.
To roll out without changing any range, set `min_duration` and `max_duration` to today's limit, then raise `max_duration` in steps.

When `max_scanned_bytes` is set, the frontend logs one `msg="scanned bytes limit"` line per tenant for each query that reaches the scanned bytes check,
with `requested_seconds`, `scanned_bytes`, `derived_seconds` or `fits`, `min_seconds`, `max_seconds`, `outcome`, and `passes_scanned_bytes`.
Queries that `max_duration` rejects are still evaluated and logged with `outcome=rejected_max_duration`, so you can see which queries a higher `max_duration` would allow.
For example, this LogQL query finds queries that a `max_duration` of 7 days would newly allow:

```logql
{namespace="tempo", container="query-frontend"} |= "scanned bytes limit" | logfmt | outcome="rejected_max_duration" | passes_scanned_bytes="true" | requested_seconds <= 604800
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
