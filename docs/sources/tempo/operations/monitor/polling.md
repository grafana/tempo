---
title: Use polling to monitor the backend status
menuTitle: Use polling to monitor backend status
description: Monitor backend status for Tempo using polling.
weight: 30
aliases:
  - /docs/tempo/operations/polling
  - ../polling
---

# Use polling to monitor the backend status

Tempo maintains knowledge of the state of the backend by polling it on regular intervals.
Backend-schedulers, backend-workers, query-frontends, and queriers with polling enabled
each maintain their own blocklist.

To reduce calls to the backend,
backend-workers perform a "full" poll that lists blocks for a tenant,
determines their state, and updates the tenant index.
When enabled, the worker ring distributes index-building jobs across backend-workers.
The ring doesn't shard tenant discovery or index reads:
each poller still processes all tenants in the backend.

Other pollers read the tenant index.
With fallback enabled, they perform a full poll if reading the index fails,
or if the index is older than the configured maximum age.

The `blocklist_poll` setting controls how often each process starts a poll.
Cycles run serially within a process,
so an overlong cycle delays subsequent updates.

The index is written in two formats: both a `gzip` compressed JSON located at `/<tenant>/index.json.gz` and a zstd compressed proto encoded object located at `/<tenant>/index.pb.zst`. Only the proto object is read, falling back to the JSON if the proto does not exist, which should only happen as part of the transition to the new format. These indexes contain an entry for every block and compacted block for the tenant.

The blocklist is eventually consistent.
When polls fit within their intervals and indexes are updated successfully,
discovery takes approximately two polling intervals:
one to build the index and one to read it.
Poll duration and failures add to this delay,
so twice `blocklist_poll` is not a guaranteed freshness bound when polling falls behind.

{{< admonition type="note" >}}
For details about configuring polling, refer to [polling configuration](../../../configuration/polling/).
For request budgeting, concurrency controls, and startup considerations with many tenants,
refer to [Scale polling across many tenants](../../../configuration/polling/#scale-polling-across-many-tenants).
For an explanation of how compaction interacts with the polling cycle and blocklist staleness, refer to [Compaction](../../compaction/).
{{< /admonition >}}

## Monitor polling with dashboards and alerts

Refer to the Jsonnet for example [alerts](https://github.com/grafana/tempo/blob/main/operations/tempo-mixin/alerts.libsonnet) and [runbook entries](https://github.com/grafana/tempo/blob/main/operations/tempo-mixin/runbook.md)
related to polling.

If you are building your own dashboards or alerts, here are a few relevant metrics:

- `tempodb_blocklist_poll_errors_total`
  A holistic metric that increments for any error with polling the blocklist. Any increase in this metric should be reviewed.
- `tempodb_blocklist_poll_duration_seconds`
  Histogram recording the length of completed polling cycles in seconds.
  A cycle that fails before applying its results isn't recorded in this histogram.
- `tempodb_blocklist_length`
  Total blocks as seen by this component.
- `tempodb_blocklist_tenant_index_errors_total`
  A holistic metric that increments for any error building the tenant index. Any increase in this metric should be reviewed.
- `tempodb_blocklist_tenant_index_builder`
  A per-tenant gauge with value `1` when a process attempts to build that tenant's index,
  including when a reader falls back to a full poll, and `0` when it reads the index.
  Each tenant needs a healthy index builder.
- `tempodb_blocklist_tenant_index_age_seconds`
  The index's age in seconds at the time it was last loaded successfully.
  This gauge doesn't increase between polls and is reset to zero when building an index,
  so use it alongside poll duration and errors to detect a stalled poller.
