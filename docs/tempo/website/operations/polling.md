---
aliases:
- /docs/tempo/v1.2.1/operations/polling/
title: Polling
weight: 8
---

# Polling

Tempo maintains knowledge of the state of the backend by polling it on regular intervals. There are currently
only two components that need this knowledge and, consequently, only two that poll the backend: compactors
and queriers. 

In order to reduce calls to the backend only a small subset of compactors actually list all blocks and build 
what's called a tenant index. The tenant index is a gzip'ed json file located at `/<tenant>/index.json.gz` containing
an entry for every block and compacted block for that tenant. This is done once every `blocklist_poll` duration.

All other compactors and all queriers then rely on downloading this file, unzipping it and using the contained list. 
Again this is done once every `blocklist_poll` duration. **NOTE** It is important that the querier `blocklist_poll` duration 
is greater than or equal to the compactor `blocklist_poll` duration. Otherwise a querier may not correctly check
all assigned blocks and incorrectly return 404.

Due to this behavior a given compactor or querier will often have an out of date blocklist. During normal operation
it will stale by at most 2x the configured `blocklist_poll`. See [configuration]({{< relref "../configuration/polling" >}})
for more information.

# Monitoring

See our jsonnet for example [alerts](https://github.com/grafana/tempo/blob/main/operations/tempo-mixin/alerts.libsonnet) and [runbook entries](https://github.com/grafana/tempo/blob/main/operations/tempo-mixin/runbook.md)
related to polling. 

If you are building your own dashboards/alerts here are a few relevant metrics:

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