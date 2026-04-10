---
title: Compaction
menuTitle: Compaction
description: How the backend scheduler and worker handle compaction and retention.
weight: 700
topicType: concept
versionDate: 2026-03-20
---

# Compaction

The backend scheduler and worker replace the legacy compactor.
Together, they handle compaction, retention, and blocklist maintenance for data in object storage.

## How it works

The backend scheduler creates jobs and assigns them to workers.
Workers connect to the scheduler via gRPC, request jobs, execute them, and report results back.
This split makes compaction horizontally scalable—you can add workers to increase throughput without changing the scheduler.

### Job types

The scheduler produces two types of jobs:

- Compaction: merges small blocks into larger ones to reduce the number of blocks queriers need to scan and improve query performance.
- Retention: deletes blocks older than the configured retention period.

### Job lifecycle

The scheduler uses providers to generate jobs.
The compaction provider periodically measures tenants and produces compaction jobs based on the blocklist.
The retention provider produces retention jobs on a schedule.

When a worker calls `Next`, the scheduler assigns an available job and persists the assignment to a local work cache.
The worker executes the job and calls `UpdateJob` with a success or failure status.
On success, the scheduler applies the results to the in-memory blocklist (for example, marking compacted blocks as removed).
The work cache is periodically flushed to object storage for crash recovery.

## Backend scheduler

The scheduler is a singleton: only one instance should run at a time.
It maintains the work cache, which tracks all active and completed jobs,
and polls object storage to keep the blocklist up to date.

The scheduler exposes an HTTP status endpoint that lists all known jobs with their status, tenant, worker assignment,
and timestamps.

```yaml
backend_scheduler:
  maintenance_interval: 1m
  backend_flush_interval: 1m
```

## Backend worker

Workers are stateless job executors. Each worker connects to the scheduler, requests a job, processes it, and reports back.
Multiple workers can run in parallel.

Workers also maintain the blocklist for all tenants.
Tenant polling is coordinated through a ring, so each worker polls a subset of tenants.
This distributes the load of scanning object storage across all workers.

Workers use a ring for tenant sharding.
The ring determines which worker is responsible for polling each tenant's blocklist.
By default the ring is disabled, meaning each worker polls all tenants without sharding.

```yaml
backend_worker:
  backend_scheduler_addr: backend-scheduler:9095
  finish_on_shutdown_timeout: 30s
```

### Graceful shutdown

When a worker receives a shutdown signal,
it has a configurable timeout (`finish_on_shutdown_timeout`) to complete the current job before being terminated.
This prevents partially completed jobs from being left in an inconsistent state.

## Key metrics

| Metric | Description |
|---|---|
| `tempodb_compaction_blocks_total` | Total blocks compacted |
| `tempodb_compaction_bytes_written_total` | Bytes written during compaction |
| `tempodb_retention_blocks_cleared_total` | Blocks deleted by retention |

## Related resources

Refer to the [compaction configuration](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#compaction) for the full list of options.
