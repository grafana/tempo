# Runbook

This document should help with remediating operational issues in Tempo.

## TempoRequestErrors
## TempoRequestLatency

Aside from obvious errors in the logs the only real lever you can pull here is scaling.  Use the Reads or Writes dashboard
to identify the component that is struggling and scale it up.

The Query path is instrumented with tracing (!) and this can be used to diagnose issues with higher latency. View the logs of
the Query Frontend, where you can find an info level message for every request. Filter for requests with high latency and view traces.

The Query Frontend allows for scaling the query path by sharding queries. There are a few knobs that can be tuned for optimum
parallelism -
- Number of shards each query is split into, configured via
    ```
    query_frontend:
        query_shards: 10
    ```
- Number of Queriers (each of these process the sharded queries in parallel). This can be changed by modifying the size of the
Querier deployment. More Queriers -> faster processing of shards in parallel -> lower request latency.

- Querier parallelism, which is a combination of a few settings:

    ```
    querier:
      max_concurrent_queries: 10
      frontend_worker:
          match_max_concurrent: true  // true by default
          parallelism: 5              // parallelism per query-frontend. ignored if match_max_concurrent is set to true

    storage:
      trace:
        pool:
          max_workers: 100
    ```

MaxConcurrentQueries defines the total number of shards each Querier processes at a given time. By default, this number will
be split between the query frontends, so if there are N query frontends, the Querier will process (Max Concurrent Queries/ N)
queries per query frontend.

Another way to increase parallelism is by increasing the size of the worker pool that queries the cache & backend blocks.

A theoretically ideal value for this config to avoid _any_ queueing would be (Size of blocklist / Max Concurrent Queries).
But also factor in the resources provided to the querier.

## TempoCompactorUnhealthy

If this occurs port-forward to 3200 on a compactor and bring up `/compactor/ring`.  Use the "Forget" button to drop any unhealthy 
compactors. An unhealthy compactor or two has no immediate impact. Long term, however, it will cause the blocklist to grow
unnecessarily long.

## TempoDistributorUnhealthy

If this occurs port-forward to 3200 on a distributor and bring up `/distributor/ring`.  Use the "Forget" button to drop any unhealthy 
distributors. An unhealthy distributor or two has virtually no impact except to slightly increase the amount of memberlist
traffic propagated by the cluster.

## TempoCompactionsFailing

Check to determine the cause for the failures.  Intermittent failures require no immediate action, because the compactors will
reattempt the compaction, and any partially written data is ignored and expected to be cleaned up automatically with bucket cleanup
rules in GCS/S3/etc.

The most common cause for a failing compaction is an OOM from compacting an extremely large trace.  The memory limit should be
increased until it is enough to get past the trace, and must remain increased until the trace goes out of retention and is
deleted, or else there is the risk of the trace causing OOMs later.  Ingester limits should be reviewed and possibly reduced.
If a block continues to cause problems and cannot be resolved it can be deleted manually.

There are several settings which can be tuned to reduce the amount of work done by compactors to help with stability or scaling:
- compaction_window - The length of time that will be compacted together by a single pod.  Can be reduced to as little as 15 or
  30 minutes.  It could be reduced even further in extremely high volume situations.
- max_block_bytes - The maximum size of an output block, and controls which input blocks will be compacted. Can be reduced to as
  little as a few GB to prevent really large compactions.
- chunk_size_bytes - The amount of (compressed) data buffered from each input block. Can be reduced to a few megabytes to buffer
  less.  Will increase the amount of reads from the backend.
- flush_size_bytes - The amount of data buffered of the output block. Can be reduced to flush more frequently to the backend.
  There are platform-specific limits on how low this can go.  AWS S3 cannot be set lower than 5MB, or cause more than 10K flushes
  per block.

## TempoIngesterFlushesFailing

How it **works**:
- Tempo ingesters flush blocks that have been completed to the backend
- If flushing fails, the ingester will keep retrying until restarted
- Blocks that have been flushed successfully will be deleted from the ingester, by default after 15m

Failed flushes could be caused by any number of different things: bad block, permissions issues, rate limiting, failing backend,...
Tempo will continue to retry sending the blocks until it succeeds, but at some point your WAL files will start failing to write due
to out of disk issues.

Known issue: this can trigger during a rollout of the ingesters, see [tempo#1035](https://github.com/grafana/tempo/issues/1035).

How to **investigate**:
- To check which ingesters are failing to flush and look for a pattern, you can use:
  ```
  sum(rate(tempo_ingester_failed_flushes_total{cluster="...", container="ingester"}[5m])) by (pod)
  ```
- Check the logs for errors

If a single block can not be flushed, this block might be corrupted. A corrupted or bad block might be missing some files or a file
might be empty, compare this block with other blocks in the WAL to verify.  
After inspecting the block, consider moving this file out of the WAL or outright deleting it. Restart the ingester to stop the retry
attempts. Removing blocks from a single ingester will not cause data loss if replication is used and the other ingesters are flushing
their blocks successfully. By default, the WAL is at `/var/tempo/wal/blocks`.

If multiple blocks can not be flushed, the local WAL disk of the ingester will be filling up. Consider increasing the amount of disk
space available to the ingester.

## TempoPollsFailing

See [Polling Issues](#polling-issues) below for general information.

If polls are failing check the component that is raising this metric and look for any obvious logs that may indicate a quick fix.
We have only seen polling failures occur due to intermittent backend issues.

## TempoTenantIndexFailures

See [Polling Issues](#polling-issues) below for general information.

If the following is being logged then things are stable (due to polling fallback) and we just need to review the logs to determine why 
there is an issue with the index and correct.
```
failed to pull bucket index for tenant. falling back to polling
```

If the following (or other errors) are being logged repeatedly then the tenant index is not being updated and more direct action is necessary.
If the core issue can not be resolved delete any tenant index that is not being updated. This will force the components to fallback to 
bucket scanning for the offending tenants.
```
failed to write tenant index
```

## TempoNoTenantIndexBuilders

See [Polling Issues](#polling-issues) below for general information.

If a cluster has no tenant index builders for a given tenant then nothing is refreshing the per tenant index. This can be dangerous
b/c other components will not be aware there is an issue as they repeatedly download a stale tenant index. In Tempo the compactors
play the role of building the tenant index. Ways to address this issue in order of preference:

- Find and forget all unhealthy compactors.
- Increase the number of compactors that attempt to build the index.
  ```
  storage:
    trace:
      blocklist_poll_tenant_index_builders: 2  # <- increase this value
  ```
- Delete tenant index files that are not being updated to force other components to fallback to scanning for these tenants. They 
  are located at `/<tenant>/index.json.gz`

## TempoTenantIndexTooOld

See [Polling Issues](#polling-issues) below for general information.

If the tenant indexes are too old we need to review the compactor logs to determine why they are failing to update. Compactors
with `tempodb_blocklist_tenant_index_builder` for the offending tenant set to 1 are expected to be creating the indexes for that
tenant and should be checked first. If no compactors are creating tenant indexes refer to [TempoNoTenantIndexBuilders](#temponotenantindexbuilders)
above.

Additionally the metric `tempodb_blocklist_tenant_index_age_seconds` can be grouped by the `tenant` label. If only one (or few) 
indexes are lagging these can be deleted to force components to manually rescan just the offending tenants.

### Polling Issues

In the case of all polling issues intermittent issues are not concerning. Sustained polling issues need to be addressed. 

Failure to poll just means that the component is not aware of the current state of the backend but will continue working
otherwise.  Queriers, for instance, will start returning 404s as their internal representation of the backend grows stale. 
Compactors will attempt to compact blocks that don't exist.

If persistent backend issues are preventing any fixes to polling then reads will start to fail, but writes will remain fine.
Alert your users accordingly!

Note that tenant indexes are built independently and an issue may only be impacting one or very few tenants. `tempodb_blocklist_tenant_index_builder`,
`tempodb_blocklist_tenant_index_age_seconds` and `tempodb_blocklist_tenant_index_errors_total` are all per-tenant metrics. If
you can isolate the impacted tenants, attempt to take targeted action instead of making sweeping changes. Your easiest lever 
to pull is to simply delete stale tenant indexes as all components will fallback to bucket listing.

