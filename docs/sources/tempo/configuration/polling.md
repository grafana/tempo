---
title: Polling
description: Learn about polling cycle configuration options in Tempo.
weight: 700
aliases:
  - /docs/tempo/configuration/polling
---

# Polling

The polling cycle is controlled by a number of configuration options detailed here.

```
storage:
    trace:
        # How often to repoll the backend for new blocks. Default is 5m
        [blocklist_poll: <duration>]

        # Number of unknown blocks to process in parallel per tenant during a full poll. Default is 50.
        [blocklist_poll_concurrency: <int>]

        # Number of tenants to poll concurrently in each process. Default is 1.
        [blocklist_poll_tenant_concurrency: <int>]

        # By default components will pull the blocklist from the tenant index. If that fails the component can
        # fallback to scanning the entire bucket. Set to false to disable this behavior. Default is true.
        [blocklist_poll_fallback: <bool>]

        # Maximum number of workers selected to build each tenant's index when the worker ring is enabled.
        # Other pollers download the index. Default 2.
        [blocklist_poll_tenant_index_builders: <int>]

        # The oldest allowable tenant index. If an index is pulled that is older than this duration,
        # the polling will consider this an error. Note that `blocklist_poll_fallback` applies here.
        # If fallback is true and a tenant index exceeds this duration, it will fall back to listing
        # the bucket contents.
        # Default 0 (disabled).
        [blocklist_poll_stale_tenant_index: <duration>]
```

When polling completes within the configured interval and indexes are updated successfully,
the [tenant index](../../operations/monitor/polling/) introduces approximately two polling intervals of discovery delay:
one for the index builder and one for the reader.
Poll duration and failures add to this delay.
The `2 * blocklist_poll` minimum values of `query_backend_after` and `compacted_block_retention` described below
are not sufficient when polling falls behind.
For a detailed explanation of how compaction interacts with the polling cycle, refer to [Compaction](../../operations/compaction/).

## Scale polling across many tenants

Each process with polling enabled discovers all tenants in the backend
and maintains its own blocklist.
Backend-workers build tenant indexes;
query-frontends, backend-schedulers, and queriers with polling enabled normally read those indexes.
A worker also reads indexes for tenants whose index-building jobs it doesn't own.
The worker ring distributes index-building jobs,
but it doesn't shard tenant discovery or index reads across these processes.
Adding replicas can therefore increase object-store traffic even when the ingest rate stays the same.

### Distribute index building

When running multiple backend-workers,
configure `backend_worker.ring.kvstore.store` to use a shared ring,
such as `memberlist`, `consul`, or `etcd`,
and verify that all workers see the same healthy members on `/backend-worker/ring`.
Refer to [Tune the consistent hash rings](../../operations/manage-advanced-systems/consistent_hash_ring/).
An empty store setting, which is the default, or `inmemory` disables sharding:
every backend-worker then builds every tenant's index.

`blocklist_poll_tenant_index_builders` controls the maximum number of selected builders **per tenant**,
not the total number of workers in the cluster.
Its default of `2` allows redundant builders.
Reducing it to `1` reduces duplicate listing and index-writing work,
but leaves each tenant with a single selected builder until ring ownership changes.
It doesn't reduce the number of processes reading that tenant's index.

### Budget requests across all pollers

Tune `blocklist_poll_tenant_concurrency` using both poll duration and the object store's request budget.
Increasing it can shorten a poll on a high-latency backend,
but the limit applies separately to every process.
It is a concurrency limit, not a cluster-wide requests-per-second limit.

For a process reading healthy tenant indexes,
a first estimate of poll duration is `tenant count * average index read time / tenant concurrency`.
For example, 10,000 tenants at 0.2 seconds per index read take about 2,000 seconds
with the default tenant concurrency of `1`,
before accounting for tenant discovery, decoding, or retries.
Increasing concurrency cannot remove the need to read each index.

When polls fit within their intervals,
estimate the average index-read rate as the sum of `indexes read per cycle / poll interval in seconds`
over all pollers.
For example, ten readers polling 10,000 tenants every five minutes require about 333 index reads per second.
This excludes index builders' listing and write requests,
block reads and writes, tenant discovery, fallback scans, and retries.
Requests can also arrive in bursts during each poll.
Leave capacity for these operations and for restarts when comparing the estimate with a provider's limits.

Use separate component configurations to tune builders and readers independently.
Increase concurrency gradually and observe both the full poll duration
and the aggregate request and throttling rates reported by the object store.
If the request budget cannot support the desired freshness,
reduce unnecessary pollers or choose a longer interval and review the data-availability settings below.
Changing concurrency alone cannot satisfy both constraints.

The other concurrency controls affect different work:

- `blocklist_poll_concurrency` limits concurrent reads of unknown block metadata **within each tenant's full poll**.
  Raising tenant concurrency also allows more of these per-tenant pools to run at once.
  It doesn't accelerate readers that only download tenant indexes.
- `storage.trace.s3.list_blocks_concurrency` controls parallel listing within a tenant on S3.
  Raising it increases listing parallelism on every concurrent full tenant poll.
- `blocklist_poll_jitter_ms` adds a random delay before reading unknown block metadata.
  It doesn't stagger tenant-index reads or enforce an account-wide request limit.

### Avoid fallback scans overwhelming the backend

With `blocklist_poll_fallback: true`,
a missing, unreadable, or stale tenant index causes a reader to list that tenant's blocks and write a replacement index.
Several readers can do this at once,
amplifying an index-builder outage or object-store throttling into more requests.
Monitor tenant-index errors and the `failed to pull bucket index for tenant. falling back to polling` log message.

Disabling fallback on readers with `blocklist_poll_fallback: false` prevents those scans,
but trades recovery for lower request volume.
A failed tenant retains its previous blocklist when the poll is within the configured tenant-failure tolerance;
if the cycle exceeds that tolerance, the entire previous blocklist remains in use.
At startup there is no previous blocklist for those tenants,
so their blocks can remain undiscoverable until an index read succeeds.
Keep index builders healthy before making this change.
If setting `blocklist_poll_stale_tenant_index`,
allow for the builders' actual update time;
an age threshold shorter than their normal cycle can repeatedly trigger fallback.

### Reduce querier polling after upgrading

Versions that support `querier.blocklist_polling` can disable polling in dedicated queriers:

```yaml
querier:
  blocklist_polling: false
```

Upgrade all components to a version supporting this setting before enabling it.
Older versions reject the setting,
and older query-frontends can send trace-by-ID jobs without the explicit block list required by a querier with polling disabled.
Refer to the [querier configuration](../#querier).
Query-frontends and backend components still maintain their blocklists;
this setting only removes the dedicated queriers' polls.

### Account for startup and delayed polls

The first poll runs synchronously during component startup.
For a backend-worker, it runs after ring initialization and before the worker enters its running state.
A slow first poll can therefore delay readiness.
Later cycles run serially within each process;
if a poll lasts longer than `blocklist_poll`, lowering the interval won't make that poll finish sooner.

Allow startup probes enough time for a full first poll
and stagger restarts to avoid simultaneous cold polls across the fleet.
Track `tempodb_blocklist_poll_duration_seconds`, tenant-index age and errors,
and the `blocklist poll complete` log message alongside the provider's request metrics.
Refer to [Use polling to monitor the backend status](../../operations/monitor/polling/)
for metric details.

## How blocks reach object storage

Block-builders consume trace data from Kafka and organize spans into blocks based on a configurable time window.
Once a block is complete, the block-builder flushes it to object storage.
After the block is written, queriers won't discover it until their next polling cycle completes.

## Handling blocklist staleness

Two mechanisms ensure data availability despite blocklist staleness:
- Live-stores cover the recent data gap
- Compacted block retention prevents premature deletion

### Live-stores cover the recent data gap

Live-stores serve recent trace data directly from Kafka, covering the window between when a block-builder flushes a new block and when queriers discover it through polling.
The query-frontend uses `query_backend_after` to control when backend storage is searched:

- Time ranges more recent than `query_backend_after` are searched only in live-stores.
- Time ranges older than `query_backend_after` are searched in backend storage.

`query_backend_after` must be set to at least twice the `blocklist_poll` duration.
A newly flushed block can take approximately `2 * blocklist_poll` to be discovered under normal polling conditions,
and longer when polls fall behind.
If `query_backend_after` is shorter than this window, the query-frontend may route requests to the backend before the block is discoverable, causing data gaps.
With the default `blocklist_poll` of 5 minutes, `query_backend_after` should be at least 10 minutes.

```
query_frontend:
    search:
        # Time after which the query-frontend starts searching backend storage. Default is 15m.
        [query_backend_after: <duration>]
```

### Compacted block retention prevents premature deletion

The `compacted_block_retention` keeps a block in object storage for a period of time after it has been compacted and the data has been merged into a new block.
This allows queriers with a stale blocklist to still access these blocks until they complete their polling cycles and have up-to-date blocklists.
At a minimum, this should be twice the configured `blocklist_poll` duration.

```
backend_worker:
  compaction:
    # How long to leave a block in the backend after it has been compacted successfully. Default is 1h
    [compacted_block_retention: <duration>]
```

Additionally, the querier `blocklist_poll` duration needs to be greater than or equal to the worker
`blocklist_poll` duration. Otherwise, a querier may not correctly check all assigned blocks and incorrectly return 404.
It is recommended to simply set both components to use the same poll duration.
