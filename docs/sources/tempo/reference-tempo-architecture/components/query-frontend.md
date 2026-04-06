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
