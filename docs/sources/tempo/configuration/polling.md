---
title: Polling
description: Learn about Tempo's polling cycle configuration options.
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

        # Maximum number of compactors that should build the tenant index. All other components will download
        # the index.  Default 2.
        [blocklist_poll_tenant_index_builders: <int>]

        # The oldest allowable tenant index. If an index is pulled that is older than this duration,
        # the polling will consider this an error. Note that `blocklist_poll_fallback` applies here.
        # If fallback is true and a tenant index exceeds this duration, it will fall back to listing
        # the bucket contents.
        # Default 0 (disabled).
        [blocklist_poll_stale_tenant_index: <duration>]
```

Due to the mechanics of the [tenant index]({{< relref "../operations/monitor/polling" >}}), the blocklist will be stale by
at most 2 times the configured `blocklist_poll` duration. There are two configuration options that need to be balanced
against the `blockist_poll` to handle this:


The ingester `complete_block_timeout` is used to hold a block in the ingester for a given period of time after
it has been flushed. This allows the ingester to return traces to the queriers while they are still unaware
of the newly flushed blocks.
```
ingester:
  # How long to hold a complete block in the ingester after it has been flushed to the backend.  Default is 15m
  [complete_block_timeout: <duration>]
```

The compactor `compacted_block_retention` is used to keep a block in the backend for a given period of time
after it has been compacted and the data is no longer needed. This allows queriers with a stale blocklist to access
these blocks successfully until they complete their polling cycles and have up to date blocklists. Like the
`complete_block_timeout`, this should be at a minimum 2x the configured `blocklist_poll` duration.

```
compactor:
  compaction:
    # How long to leave a block in the backend after it has been compacted successfully.  Default is 1h
    [compacted_block_retention: <duration>]
```

Additionally, the querier `blocklist_poll` duration needs to be greater than or equal to the compactor
`blocklist_poll` duration. Otherwise, a querier may not correctly check all assigned blocks and incorrectly return 404.
It is recommended to simply set both components to use the same poll duration.
