# Runbook

This document should help with remediation of operational issues in Tempo.

### Trace Lookup Failures

If trace lookups are fail with the error: `error querying store in Querier.FindTraceByID: queue doesn't have room for <xyz> jobs`, this
means that the number of blocks has exceeded the queriers' ability to check them all.  This can be caused by an increase in
the number of blocks, or if the number of queriers was reduced.

Check the following metrics and data points:
- Metric: `tempodb_blocklist_length` - Look for a recent increase in blocks
- The number of queriers
- The queue_depth setting in the queriers:
    ```
    storage:
      trace:
        pool:
          queue_depth: xyz (default 10000)
    ```

The queue won't have room when the number of blocks per querier exceeds the queue_depth:
  (blocklist_length / number of queriers) > queue_depth

Consider the following resolutions:
- Increase the number of queriers
- Increase the queue_depth size to do more work per querier
- Adjust compaction settings to reduce the number of blocks

## TempoDistributorUnhealthy

This can happen when we have unhealthy distributor sticking around in the ring.

If this occurs access the [ring page](https://grafana.com/docs/tempo/latest/operations/consistent_hash_ring/) at `/distributor/ring`.
Use the "Forget" button to forget and remove any unhealthy distributors from the ring. An unhealthy distributor or two has virtually no impact except to slightly
increase the amount of memberlist traffic propagated by the cluster.

## TempoLiveStoreUnhealthy

This can happen when we have unhealthy livestore sticking around in the ring.

If this occurs, access the [ring page](https://grafana.com/docs/tempo/latest/operations/consistent_hash_ring/) at `/live-store/ring`.
Use the "Forget" button to forget and remove any unhealthy livestore from the ring.

## TempoMetricsGeneratorUnhealthy

This can happen when we have unhealthy metrics-generators sticking around in the ring.

If this occurs, access the [ring page](https://grafana.com/docs/tempo/latest/operations/consistent_hash_ring/) at `/metrics-generator/ring`.
Use the "Forget" button to forget and remove any unhealthy metrics-generators from the ring.

## TempoCompactionsFailing

Check to determine the cause for the failures.  Intermittent failures require no immediate action, because the backend scheduller will
reattempt the compaction, and any partially written data is ignored and expected to be cleaned up automatically with bucket cleanup
rules in GCS/S3/etc.

The most common cause for a failing compaction is an OOM from compacting an extremely large trace.  The memory limit should be
increased until it is enough to get past the trace, and must remain increased until the trace goes out of retention and is
deleted, or else there is the risk of the trace causing OOMs later. Ingestion limits should be reviewed and possibly reduced.
If a block continues to cause problems and cannot be resolved it can be deleted manually.


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
b/c other components will not be aware there is an issue as they repeatedly download a stale tenant index. In Tempo the backend scheduler and backend workers
play the role of building the tenant index. Ways to address this issue in order of preference:

- Increase the number of backend workers that attempt to build the index.
  ```
  storage:
    trace:
      blocklist_poll_tenant_index_builders: 2  # <- increase this value
  ```
- Delete tenant index files that are not being updated to force other components to fallback to scanning for these
  tenants. They are located at `/<tenant>/index.json.gz` in the configured long-term/object-storage backend.

## TempoTenantIndexTooOld

See [Polling Issues](#polling-issues) below for general information.

If the tenant indexes are too old we need to review the backend scheduler/worker logs to determine why they are failing to update. Workers
with `tempodb_blocklist_tenant_index_builder` for the offending tenant set to 1 are expected to be creating the indexes for that
tenant and should be checked first. If no woerkers are creating tenant indexes refer to [TempoNoTenantIndexBuilders](#temponotenantindexbuilders)
above.

Additionally the metric `tempodb_blocklist_tenant_index_age_seconds` can be grouped by the `tenant` label. If only one (or few)
indexes are lagging these can be deleted to force components to manually rescan just the offending tenants.

### Polling Issues

In the case of all polling issues intermittent issues are not concerning. Sustained polling issues need to be addressed.

Failure to poll just means that the component is not aware of the current state of the backend but will continue working
otherwise.  Queriers, for instance, will start returning 404s as their internal representation of the backend grows stale.
Backend workers will attempt to compact blocks that don't exist.

If persistent backend issues are preventing any fixes to polling then reads will start to fail, but writes will remain fine.
Alert your users accordingly!

Note that tenant indexes are built independently and an issue may only be impacting one or very few tenants. `tempodb_blocklist_tenant_index_builder`,
`tempodb_blocklist_tenant_index_age_seconds` and `tempodb_blocklist_tenant_index_errors_total` are all per-tenant metrics. If
you can isolate the impacted tenants, attempt to take targeted action instead of making sweeping changes. Your easiest lever
to pull is to simply delete stale tenant indexes as all components will fallback to bucket listing. The tenant index is located at:

```
/<tenant>/index.json.gz
```

## TempoBlockListRisingQuickly

The block list needs to remain under control to keep query performance acceptable.  If the block list is rising too quickly, this might indicate the backend workers are under scaled.  Add more workers until the block list is back under control and holding mostly steady.


### TempoBadOverrides

Fix the overrides!  Overrides are loaded by the distributors so hopefully there is
some meaningful logging there.


## TempoCompactionTooManyOutstandingBlocks

This alert fires when there are too many blocks to be compacted for a long period of time.
The alert does not require immediate action, but is a symptom that compaction is underscaled
and could affect the read path in particular.

How to fix:

Compaction's bottleneck is most commonly CPU time, so adding more backend workers is the most effective measure.

After compaction has been scaled out, it'll take a time for backend workers to catch
up with their outstanding blocks.
Take a look at `tempodb_compaction_outstanding_blocks` and check if blocks start
going down. If not, further scaling may be necessary.

Since the number of blocks is elevated, it may also be necessary to review the queue-related
settings to prevent [trace lookup failures](#trace-lookup-failures).

## TempoPartitionLag

This alert fires when a Kafka partition in a consumer group is lagging behind the latest offset by a significant amount of time.

### Troubleshooting

1. Check the general health of the affected component (block-builder or metrics-generator):
   - Review logs for errors or warnings related to Kafka consumption
   - Check if the component is experiencing high CPU or memory usage
   - Look for any unusual patterns in processing time or error rates

2. Check the health of the Kafka cluster:
   - Verify broker health and connectivity
   - Check if there are any network issues between Tempo and Kafka
   - Examine Kafka metrics for unusual patterns (high produce rate, throttling, etc.)

3. Possible resolutions:
   - For the metric generators scale up the consumer group by adding more instances
   - Scale up the block-builder instances to match the live-store partitions.
   - For the block-builder consider to reduce the cycle-time
   - Increase resources (CPU/memory) for the consumer instances
   - Check for and fix any bottlenecks in the processing pipeline
   - If the lag is temporary due to a spike in traffic, monitor to see if it recovers


## TempoBlockBuildersPartitionsMismatch

This alert fires when more than 1 active or inactive partition has been un-owned by a block-builder for more than 10 minutes.

### How it works

Each block-builder is configured to consume from one or more Kafka partitions.

The distributor uses the partition ring to determine which partitions are active and where each trace should be sent.


### How to investigate it

1. Ensure that all the blockbuilders are running and not crashlooping.
2. Look what partition is missing. This metric will return the assigned ones and their state:
   `tempo_block_builder_owned_partitions{namespace="tempo-dev-01"}`
3. Check is that partition is assigned to any of the block-builder instances. Look for any error.


## TempoLiveStoresPartitionsUnowned

This alert fires when one or more Kafka partitions have no active live-store instances consuming from them for more than 10 minutes.

### How it works

- Live-stores consume traces from Kafka partitions and serve them on the read path
- Each active partition should have at least one live-store instance consuming from it across all zones
- The alert compares the total number of active/inactive partitions in the ring against the count of partitions that have at least one active consumer
- If there are more partitions than consumers, some partitions are unowned

### Impact

- **Read path degradation**: Queries for recent traces in unowned partitions will fail or return incomplete results
- **Critical**: This is a data availability issue that needs immediate attention

### How to investigate

1. Check which partitions are unowned by comparing the partition ring to active consumers:
   ```
   tempo_partition_ring_partitions{name="livestore-partitions", state=~"Active|Inactive"}
   tempo_live_store_partition_owned{}
   ```

2. Check if live-store pods are running and healthy:
   ```bash
   kubectl -n <namespace> get pods -l name=live-store
   kubectl -n <namespace> logs -l name=live-store --tail=100
   ```

3. Look for errors in the logs related to live-store pod health

### How to fix

1. **If live-store pods are down or crashlooping:**
   - Check pod status and logs for errors
   - If OOM, consider scaling resources
   - Restart unhealthy pods if needed

2. **If live-store pods are running but not claiming partitions:**
   - Check logs for errors

## TempoLiveStoreZoneSeverelyDegraded

This alert fires when a zone owns 60% fewer partitions than other zones,
indicating that a large portion of live-store pods in that zone are unhealthy.

### How to investigate

1. Check pod health in the affected zone:
   ```bash
   kubectl -n <namespace> get pods -l name=live-store -o wide
   ```

2. Review health metrics to see partition distribution across zones:
   ```
   tempo_live_store_partition_owned
   ```

3. Check logs for errors in the affected zone's pods:
   ```bash
   kubectl -n <namespace> logs -l name=live-store --tail=100
   ```

4. Look at Kubernetes events for any issues (node problems, scheduling failures, etc.):
   ```bash
   kubectl -n <namespace> get events --sort-by='.lastTimestamp'
   ```

### How to fix

- Restart unhealthy pods in the affected zone
- Check for node-level issues if multiple pods are down
- Verify zone has sufficient resources to run live-store pods

## TempoDistributorUsageTrackerErrors

The Tempo distributor is encountering errors when usage tracker/cost attribution for a tenant. This is caused by a misconfigured usage tracker configuration for the affected tenant.

This alert means that tenant's cost attribution is not working or is incorrect due to this misconfiguration.

### Troubleshooting Steps

1. Check the alert labels to identify the affected (alerting metric has the tenant, reason, cluster, and namespace labels).
2. Review the tenant's usage tracker configuration for issues
3. Check distributor logs for detailed error messages and more details of the reason.
4. If the configuration is incorrect, work with the impacted tenant to fix their usage tracker settings.


## TempoMetricsGeneratorProcessorUpdatesFailing

The metrics-generator contains processors to convert spans into metrics. They can be enabled/disabled dynamically per
tenant. The metrics-generator periodically reads the overrides file and updates the active processors accordingly.

Updating the processors might fail if an invalid processor is configured (i.e. a typo) or if creating and registering
the new processor failed. If the update failed, the metrics-generator will keep retrying. This process is per tenant and
failed updates to a single tenant should not impact others.

How to investigate:

1. Check the logs.
2. Verify if there were any recent changes to the overrides file.

## TempoMetricsGeneratorServiceGraphsDroppingSpans

The service graphs processor buffers spans internally. If the buffer is full, spans will be dropped. This will result in
incomplete service graphs metrics.

How to fix:

1. Increase buffer size or increase the amount of workers.

## TempoMetricsGeneratorCollectionsFailing

The metrics-generator has a registry that keeps track of all the active counters. On a regular interval the state of
these counters will be collected and appended as samples onto the WAL. These samples are sent out using remote write.

If a collection fails, the samples will be dropped and the metrics for this tenant will be interrupted.

How to investigate:

1. Check the logs to find out why the collection failed. Search for the phrase "collecting metrics failed".

## TempoMemcachedErrorsElevated

This alert fires when memcached request errors exceed 20% for a role in a namespace/cluster.

Likely causes include an overloaded memcached tier, network issues, or backend store timeouts surfacing as 5xx errors.

### How to investigate

1. Check memcached pod health and recent restarts.
2. Review memcached logs for 5xx responses or connection errors.
3. Compare request rate and cache hit ratio to see if the tier is saturated.
4. Look for backend store or network timeouts that could be causing request failures.

### How to fix

- Scale memcached for the affected role (more replicas or resources).
- Increase memcached capacity limits if eviction or memory pressure is high.
- Address backend or network issues if errors correlate with timeouts upstream.

## TempoBackendSchedulerJobsFailureRateHigh

This alert fires when the failure rate of backend scheduler jobs exceeds alert threshold.

This is a strong signal that job processing is having issues and job are failing to schedule.

- Inspect logs of the backend scheduler component for errors
- Identify job types and tenants contributing to failures

If the failure rate stays high, the blocklist might grow because compaction might not happen due the failed job scheduling.


## TempoBackendSchedulerRetryRateHigh

This alert fires when a high number of jobs are being retried by workers,
indicating execution instability or transient failures or possible issues with backend object stores

- Check logs for tenant and job types that are being retried
- Look at logs to see if the backend object store is having issues or not

## TempoBackendSchedulerCompactionEmptyJobRateHigh

This alert fires when a high number of jobs received by the backend scheduler
are empty. This can happen when the backoff for the compaction provider is too
low and the outstanding blocklist is low, meaning that there is not enough work
to do.

- Check logs for backend scheduler component for errors
- Check the blocklist and compare to the outstanding block list

## TempoBackendWorkerBadJobsRateHigh

This alert fires when backend workers receive an elevated number of bad jobs.

This should not fire under normal operations and is most likey due bad jobs being generated by scheduler

This can also happen due to a bug or version/schema mismatch where a new version of scheduler is generating
jobs while the worker is on old version/schema.

Audit the jobs that are marked as bad and fix them to resolve this alert.


## TempoBackendWorkerCallRetriesHigh

This alert fires when backend workers retry calls frequently to the scheduler.

This can mean multiple things:
- Scheduler is down or unreacbale from workers
- scheduler is rate limiting the workers
- scheduler is having issues handing out job or failing internallt due to outage/issues on backend object store.

Look at the logs for scheduler and worker and fix the issues.

## TempoVultureHighErrorRate

This alert fires when Tempo vulture detects a high error rate (above the configured threshold) while validating write or read paths. It indicates there are problems with trace processing or storage.
