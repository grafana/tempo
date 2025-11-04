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

Tempo maintains knowledge of the state of the backend by polling it on regular intervals. There are only
only a few components that need this knowledge: compactors, schedulers, workers, queriers and query-frontends.

To reduce calls to the backend, only the compactors and workers perform a "full" poll against the backend and update the tenant indexes. This process lists all blocks for a given tenant and determines their state. The ring is used to split the work of writing the tenant indexes for all tenants.

The remaining components will only read the tenant index, and fall back to a full poll only if the index is too far out of date.

For both the read and write of the tenant index, the update is performed once each `blocklist_poll` duration.

The index is written in two formats: both a `gzip` compressed JSON located at `/<tenant>/index.json.gz` and a zstd compressed proto encoded object located at `/<tenant/index.pb.zst`. Only the proto object is read, falling back to the JSON if the proto does not exist, which should only happen as part of the transition to the new format. These indexes contain an entry for every block and compacted block for the tenant.

Due to this behavior, a given poller will always have an out-of-date blocklist.
During normal operation, the index will be stale by at most twice the configured `blocklist_poll`. An index which is out of date by greater than the `blocklist_poll` duration and will affect which blocks are queryable, and poller configuration adjustments may need to be made in order to keep up with the size of the blocklist.

{{< admonition type="note" >}}
For details about configuring polling, refer to [polling configuration](../../../configuration/polling/).
{{< /admonition >}}

## Monitor polling with dashboards and alerts

Refer to the Jsonnet for example [alerts](https://github.com/grafana/tempo/blob/main/operations/tempo-mixin/alerts.libsonnet) and [runbook entries](https://github.com/grafana/tempo/blob/main/operations/tempo-mixin/runbook.md)
related to polling.

If you are building your own dashboards or alerts, here are a few relevant metrics:

- `tempodb_blocklist_poll_errors_total`
  A holistic metric that increments for any error with polling the blocklist. Any increase in this metric should be reviewed.
- `tempodb_blocklist_poll_duration_seconds`
  Histogram recording the length of time in seconds to poll the entire blocklist.
- `tempodb_blocklist_length`
  Total blocks as seen by this component.
- `tempodb_blocklist_tenant_index_errors_total`
  A holistic metrics that indcrements for any error building the tenant index. Any increase in this metric should be reviewed.
- `tempodb_blocklist_tenant_index_builder`
  A gauge that has the value 1 if this compactor is attempting to build the tenant index and 0 if it is not. At least one compactor
  must have this value set to 1 for the system to be working.
- `tempodb_blocklist_tenant_index_age_seconds`
  The age of the last loaded tenant index. now() minus this value indicates how stale this components view of the blocklist is.
