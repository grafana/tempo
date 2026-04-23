---
title: Compaction
description: Learn how Tempo compacts blocks in the backend and how to configure the compaction process.
weight: 200
---

# Compaction

Tempo stores trace data as immutable blocks in object storage.
Over time, many small blocks accumulate from frequent flushes.
Compaction merges groups of small blocks into fewer, larger blocks, reducing storage costs and improving query performance by limiting the number of blocks a querier must scan.

## Block states

Every block has one of two states, signaled by which metadata file is present in the backend:

- **Live** (`meta.json`): the block is queryable and included in query plans.
- **Compacted** (`meta.compacted.json`): the block's data has been merged into a new block. The file is renamed from `meta.json` to `meta.compacted.json` at the time of compaction.

Blocks are never deleted immediately after compaction.
Because the blocklist is polled periodically, queriers may not yet know about the new merged block when the old blocks are first marked compacted.
The `compacted_block_retention` window keeps the old blocks readable until all pollers have caught up.

## Blocklist and poller

Tempo components rely on an in-memory blocklist to know which blocks exist in object storage.
They do not scan object storage directly on each request.
The blocklist is maintained by the poller, a background process that reads the backend on a fixed interval controlled by `blocklist_poll` (default 5 minutes).
The poller is how compaction results become visible to queriers and other components: newly merged blocks appear in the blocklist and compacted blocks are removed.
A component can only serve data from blocks the poller has discovered. If the poller falls behind or fails, the component's view of the backend becomes incomplete.

The poller reads a pre-built tenant index from the backend rather than scanning all objects on each cycle.
If the tenant index is missing or older than `blocklist_poll_stale_tenant_index`, the component falls back to scanning the backend directly.
Setting `blocklist_poll_fallback: true` (the default) enables this fallback; disabling it means a stale or missing index causes the poll to fail rather than recover.

Because the poller runs on an interval, the blocklist is always slightly out of date.
In a healthy system, the blocklist will be stale by at most twice the configured `blocklist_poll` duration.
This is because a writer may update the backend at any point within a polling cycle, and the reader may have just started its own cycle at that same moment.

## Timing requirements

The 2x staleness bound drives two concrete configuration requirements.

### query_backend_after must be at least 2x blocklist_poll

The query-frontend uses `query_backend_after` to decide where to search:

- Traces more recent than `query_backend_after` are served from live-stores only.
- Traces older than `query_backend_after` are served from the backend.

A newly flushed block will not appear in the querier's blocklist until up to `2 * blocklist_poll` after it was written.
During that window, the block would be missed if the query-frontend routed the request to the backend.
Setting `query_backend_after` to at least `2 * blocklist_poll` ensures that such blocks are still covered by live-stores and are not yet expected to appear in the backend.

With the default `blocklist_poll` of 5 minutes, `query_backend_after` should be at least 10 minutes.
The default of 15 minutes for search provides a comfortable margin.

### compacted_block_retention must be at least 2x blocklist_poll

When a block is compacted, its data is merged into a new block and the old block is marked compacted.
A querier with a stale blocklist may still reference the old block until it completes its next poll cycle.
Setting `compacted_block_retention` to at least `2 * blocklist_poll` ensures those queriers can still read the old block before it is deleted.
The default of 1 hour is well above the minimum for typical configurations.

## Block selection

Before any compaction work begins, the block selector determines which blocks to compact together.

The time window block selector groups blocks by their time range and compaction level.
Blocks are only compacted with other blocks that share the same compaction window, data encoding, format version, and dedicated column configuration.

The selector divides the blocklist into two zones based on the **active window** (the most recent 24 hours):

- **Inside the active window**: blocks are grouped by compaction level, then by time window. Lower compaction levels are prioritized, and within each group the smallest blocks are selected first.
- **Outside the active window**: blocks are grouped by time window only, ignoring compaction level. This allows older data to be consolidated regardless of how many times it has already been compacted.

The selector only returns a group for compaction if it contains at least the minimum number of input blocks (default 2).
Each input set is also bounded by `max_block_bytes` (default 100 GB).
If a full group would exceed this limit, the selector picks the largest contiguous subset of blocks that fits; any remaining blocks in the group are eligible in subsequent compaction runs.

The number of input blocks per compaction job is configurable via `min_input_blocks` and `max_input_blocks`, defaulting to a minimum of 2 and a maximum of 4.

## How compaction works

Compaction is handled by the backend scheduler and backend workers.
The scheduler is a singleton that assigns compaction jobs to a pool of workers.
Workers request jobs from the scheduler and report back when complete; the scheduler tracks active jobs and avoids handing out overlapping work.

The scheduler uses a priority queue to choose which tenant to work on next.
Outstanding blocks are those the block selector has identified as eligible for compaction -- blocks that meet the window, encoding, and minimum-group-size requirements.
Tenants are ranked by the sum of their total blocklist length and their outstanding block count, so tenants with more overall backend work receive jobs first.
A tenant with no outstanding blocks receives a priority of zero and is not scheduled.

The scheduler processes one tenant at a time, issuing jobs until the tenant's eligible blocks are exhausted or the per-tenant job limit (`max_jobs_per_tenant`, default 1000) is reached, then moves to the next tenant.
This cap is the mechanism that prevents a single busy tenant from monopolizing the workers.

Within a tenant, the block selector prioritizes lower compaction levels first.
Higher-level blocks are still compacted, but after lower-level work is cleared.

The compaction provider waits for a poll notification before selecting new work, preventing idle spin when no blocks are eligible.
A minimum cycle interval (`min_cycle_interval`, default 30 seconds) rate-limits tenant prioritization when the queue is empty.

### Autoscaling

Outstanding block counts are measured for all tenants on a periodic ticker (`measure_interval`, default 1 minute), giving an aggregate view of total backend work.
Autoscaling thresholds should be set against this aggregate total rather than per-instance averages.
For a reference KEDA autoscaling configuration driven by outstanding block metrics, see `operations/jsonnet/microservices/autoscaling.libsonnet`.

## Configuration reference

For a full configuration reference, refer to the [configuration documentation](../configuration/).

The following fields are most relevant to compaction behavior.

### Polling interval (`storage.trace`)
```yaml
storage:
  trace:
    # How often to poll the backend for new blocks. Default: 5m.
    blocklist_poll: 5m
```

### Compaction settings (`backend_worker.compaction`)
```yaml
backend_worker:
  compaction:
    # Size of the time window used to group blocks. Default: 1h.
    compaction_window: 1h

    # Maximum size of a compacted output block in bytes. Default: 100GB.
    max_block_bytes: 107374182400

    # How long to keep a block after compaction. Must be >= 2x blocklist_poll. Default: 1h.
    compacted_block_retention: 1h
```

### Query cutoff (must be >= 2x blocklist_poll)
```yaml
query_frontend:
  search:
    # Default: 15m.
    query_backend_after: 15m
```

### Scheduler compaction (`backend_scheduler.provider.compaction`)
```yaml
backend_scheduler:
  provider:
    compaction:
      # Maximum jobs to issue per tenant before switching. Default: 1000.
      max_jobs_per_tenant: 1000

      # Minimum number of input blocks per job. Default: 2.
      min_input_blocks: 2

      # Maximum number of input blocks per job. Default: 4.
      max_input_blocks: 4

      # Minimum time between tenant selection cycles. Default: 30s.
      min_cycle_interval: 30s
```

{{< admonition type="note" >}}
The `backend_worker.compaction` configuration block contains two fields that are not used by the current scheduled compaction path: `max_time_per_tenant` and `compaction_cycle`.
These fields were used by the ring-based compaction loop, which has been removed.
They are accepted by the configuration parser but have no effect.
{{< /admonition >}}

## See also

- [Polling configuration](../configuration/polling/)
- [Monitor backend polling](./monitor/polling/)
