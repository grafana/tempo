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

The scheduler produces three types of jobs:

- Compaction: merges small blocks into larger ones to reduce the number of blocks queriers need to scan and improve query performance.
- Retention: deletes blocks older than the configured retention period.
- Redaction: rewrites blocks to remove matching trace data from object storage.

### Job lifecycle

The scheduler uses providers to generate jobs.
Each provider runs independently and feeds jobs into a shared channel.

- The compaction provider periodically measures tenants and produces compaction jobs based on the blocklist.
- The retention provider produces retention jobs on a schedule.
- The redaction provider drains a persistent queue of pending redaction requests. The scheduler's rescan logic handles waiting for any compaction jobs that were active at submission time to complete before the rewritten blocks become eligible for querying.

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

## Scheduler status API

The backend scheduler exposes an HTTP endpoint that shows the current state of all jobs:

```
GET /status/backendscheduler
```

The response is a plain-text table with two sections:

- Active Jobs: all jobs in the scheduler work cache, sorted by creation time. This includes jobs in any state -- use the `status` column to interpret each row. A non-empty `worker` field indicates the job is currently assigned to a worker.
- Pending Jobs: redaction jobs in the pending queue. Some may already be eligible to run; others may still be waiting for the rescan or compaction preconditions to clear.

This endpoint is useful for diagnosing stalled jobs, verifying that workers are consuming work, and checking whether a redaction request has been processed.

## Key metrics

| Metric | Description |
|---|---|
| `tempodb_compaction_blocks_total` | Blocks compacted |
| `tempodb_compaction_bytes_written_total` | Bytes written during compaction |
| `tempodb_retention_marked_for_deletion_total` | Blocks marked for deletion by retention |
| `tempodb_retention_deleted_total` | Blocks deleted by retention |
| `tempo_backend_scheduler_jobs_created_total` | Jobs created |
| `tempo_backend_scheduler_jobs_completed_total` | Jobs completed successfully |
| `tempo_backend_scheduler_jobs_failed_total` | Jobs that failed |
| `tempo_backend_scheduler_jobs_active` | Jobs currently assigned to a worker |
| `tempo_backend_scheduler_job_duration_seconds` | Job execution duration histogram |
| `tempodb_blocklist_length` | Number of live blocks per tenant; high values indicate compaction is falling behind |
| `tempodb_compaction_outstanding_blocks` | Outstanding blocks awaiting compaction per tenant; the primary autoscaling signal |

Most scheduler job metrics carry `tenant` and `job_type` labels; `tempo_backend_scheduler_job_duration_seconds` carries only `job_type`.
The `job_type` label uses protobuf enum string values: `JOB_TYPE_COMPACTION`, `JOB_TYPE_RETENTION`, and `JOB_TYPE_REDACTION`.
The duration histogram measures elapsed time from job creation to completion, not execution time alone.

## Monitoring

The Tempo mixin ships a pre-built Grafana dashboard, **Tempo - Backend Work**, that covers:

- Blocklist length and poll duration
- Active, completed, failed, and retried job counts
- Compaction throughput (objects written, bytes written, blocks compacted)
- Outstanding blocks per tenant
- CPU and memory for both the backend scheduler and backend workers
- A backend-worker autoscaling panel

To use the dashboard, install the Tempo mixin from `operations/tempo-mixin/` and import the generated dashboard into your Grafana instance.

## Related resources

- [Compaction operations](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/compaction/) for timing requirements and block selection details.
- [Configuration reference](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#backend-scheduler) for the full list of options.
