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

        # Number of blocks to process in parallel during polling. Default is 50.
        [blocklist_poll_concurrency: <int>]

        # By default components will pull the blocklist from the tenant index. If that fails the component can
        # fallback to scanning the entire bucket. Set to false to disable this behavior. Default is true.
        [blocklist_poll_fallback: <bool>]

        # Maximum number of workers that should build the tenant index. All other components will download
        # the index. Default 2.
        [blocklist_poll_tenant_index_builders: <int>]

        # The oldest allowable tenant index. If an index is pulled that is older than this duration,
        # the polling will consider this an error. Note that `blocklist_poll_fallback` applies here.
        # If fallback is true and a tenant index exceeds this duration, it will fall back to listing
        # the bucket contents.
        # Default 0 (disabled).
        [blocklist_poll_stale_tenant_index: <duration>]
```

Due to the mechanics of the [tenant index](../../operations/monitor/polling/), the blocklist will be stale by
at most twice the configured `blocklist_poll` duration.

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
