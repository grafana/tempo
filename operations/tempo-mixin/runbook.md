# Runbook

This document should help with remediation of operational issues in Tempo.

## Trace Lookup Failures

If trace lookups fail with the error: `error querying store in Querier.FindTraceByID: queue doesn't have room for <xyz> jobs`, this
means that the number of blocks has exceeded the queriers' ability to check them all. This can be caused by an increase in
the number of blocks, or if the number of queriers was reduced.

The queue won't have room when the number of blocks per querier exceeds the queue depth:

`(blocklist_length / number of queriers) > queue_depth`

Check the `queue_depth` setting in the queriers:

```
storage:
  trace:
    pool:
      queue_depth: xyz (default 10000)
```

Consider the following resolutions:
- Increase the number of queriers
- Increase the queue depth size to do more work per querier
- Adjust compaction settings to reduce the number of blocks

### Quick checks
- Metric/query: `tempodb_blocklist_length`
- Metric/query: `count(tempo_build_info{container="querier"})`
- Log query: `{container="querier"} |= "queue doesn't have room for"`

## TempoDistributorUnhealthy

This can happen when we have unhealthy distributors sticking around in the ring.

If this occurs access the [ring page](https://grafana.com/docs/tempo/latest/operations/consistent_hash_ring/) at `/distributor/ring`.
Use the "Forget" button to forget and remove any unhealthy distributors from the ring. An unhealthy distributor or two has virtually no impact except to slightly
increase the amount of memberlist traffic propagated by the cluster.


### Quick checks
- Metric/query: `tempo_ring_members{state="Unhealthy", name="distributor"}`
- Metric/query: `increase(kube_pod_container_status_restarts_total{container="distributor"}[10m])`
- Log query: `{container="distributor"}`

## TempoLiveStoreUnhealthy

This can happen when we have unhealthy live-stores sticking around in the ring.

If this occurs, access the [ring page](https://grafana.com/docs/tempo/latest/operations/consistent_hash_ring/) at `/live-store/ring`.
Use the "Forget" button to forget and remove any unhealthy live-store instances from the ring.


### Quick checks
- Metric/query: `tempo_ring_members{state="Unhealthy", name="live-store"}`
- Metric/query: `tempo_live_store_partition_owned{}`
- Log query: `{container="live-store"}`

## TempoMetricsGeneratorUnhealthy

This can happen when we have unhealthy metrics-generators sticking around in the ring.

If this occurs, access the [ring page](https://grafana.com/docs/tempo/latest/operations/consistent_hash_ring/) at `/metrics-generator/ring`.
Use the "Forget" button to forget and remove any unhealthy metrics-generators from the ring.


### Quick checks
- Metric/query: `tempo_ring_members{state="Unhealthy", name="metrics-generator"}`
- Metric/query: `max by (partition) (tempo_ingest_group_partition_lag_seconds{container="metrics-generator"})`
- Log query: `{container="metrics-generator"}`

## TempoCompactionsFailing

Check to determine the cause for the failures. Intermittent failures require no immediate action, because the backend scheduler will
reattempt the compaction, and any partially written data is ignored and expected to be cleaned up automatically with bucket cleanup
rules in GCS/S3/etc.

The most common cause for a failing compaction is an OOM from compacting an extremely large trace. The memory limit should be
increased until it is enough to get past the trace, and must remain increased until the trace goes out of retention and is
deleted, or else there is the risk of the trace causing OOMs later. Ingestion limits should be reviewed and possibly reduced.
If a block continues to cause problems and cannot be resolved it can be deleted manually.

In the current backend-scheduler/backend-worker flow, the scheduler exposes backlog metrics such as `tempodb_compaction_outstanding_blocks`, while backend-workers perform the compaction work. Use scheduler metrics to understand queue depth and worker metrics/logs to identify resource pressure or job failures.

### Quick checks
- Metric/query: `sum by (cluster, namespace) (increase(tempodb_compaction_errors_total[1h]))`
- Metric/query: `sum by (tenant) (tempodb_compaction_outstanding_blocks{container="backend-scheduler"})`
- Metric/query: `max by (pod) (container_memory_working_set_bytes{container="backend-worker"})`
- Log query: `{container=~"backend-worker|backend-scheduler"} |= "compaction"`

## TempoPollsFailing

See [Polling Issues](#polling-issues) below for general information.

If polls are failing check the component that is raising this metric and look for any obvious logs that may indicate a quick fix.
We have only seen polling failures occur due to intermittent backend issues.


### Quick checks
- Metric/query: `sum by (job) (increase(tempodb_blocklist_poll_errors_total[1h]))`
- Metric/query: `max by (job) (tempodb_blocklist_length)`
- Log query: `{container=~"querier|query-frontend|backend-worker|backend-scheduler"} |= "poll"`

## TempoTenantIndexFailures

See [Polling Issues](#polling-issues) below for general information.

If the following is being logged then things are stable (due to polling fallback) and we just need to review the logs to determine why
there is an issue with the index and correct it.

```
failed to pull bucket index for tenant. falling back to polling
```

If the following (or other errors) are being logged repeatedly then the tenant index is not being updated and more direct action is necessary.
If the core issue cannot be resolved delete any tenant index that is not being updated. This will force the components to fallback to
bucket scanning for the offending tenants.

```
failed to write tenant index
```


### Quick checks
- Metric/query: `sum by (tenant) (increase(tempodb_blocklist_tenant_index_errors_total[1h]))`
- Metric/query: `max by (tenant) (tempodb_blocklist_tenant_index_age_seconds)`
- Log query: `{container=~"backend-worker|backend-scheduler|querier|query-frontend"} |= "tenant index"`

## TempoNoTenantIndexBuilders

See [Polling Issues](#polling-issues) below for general information.

If a cluster has no tenant index builders for a given tenant then nothing is refreshing the per-tenant index. This can be dangerous
because other components will not be aware there is an issue as they repeatedly download a stale tenant index. In Tempo the backend scheduler and backend workers
play the role of building the tenant index. Ways to address this issue in order of preference:

- Increase the number of backend workers that attempt to build the index.
  ```
  storage:
    trace:
      blocklist_poll_tenant_index_builders: 2  # <- increase this value
  ```
- Delete tenant index files that are not being updated to force other components to fallback to scanning for these
  tenants. They are located at `/<tenant>/index.json.gz` in the configured long-term/object-storage backend.


### Quick checks
- Metric/query: `sum by (tenant) (tempodb_blocklist_tenant_index_builder)`
- Metric/query: `max by (tenant) (tempodb_blocklist_tenant_index_age_seconds)`
- Log query: `{container=~"backend-worker|backend-scheduler"} |= "tenant index"`

## TempoTenantIndexTooOld

See [Polling Issues](#polling-issues) below for general information.

If the tenant indexes are too old we need to review the backend scheduler/worker logs to determine why they are failing to update. Workers
with `tempodb_blocklist_tenant_index_builder` for the offending tenant set to 1 are expected to be creating the indexes for that
tenant and should be checked first. If no workers are creating tenant indexes refer to [TempoNoTenantIndexBuilders](#temponotenantindexbuilders)
above.

Additionally the metric `tempodb_blocklist_tenant_index_age_seconds` can be grouped by the `tenant` label. If only one (or few)
indexes are lagging these can be deleted to force components to manually rescan just the offending tenants.

### Polling Issues

In the case of all polling issues intermittent issues are not concerning. Sustained polling issues need to be addressed.

Failure to poll just means that the component is not aware of the current state of the backend but will continue working
otherwise. Queriers, for instance, will start returning 404s as their internal representation of the backend grows stale.
Backend workers will attempt to compact blocks that don't exist.

If persistent backend issues are preventing any fixes to polling then reads will start to fail, but writes will remain fine.
Alert your users accordingly.

Note that tenant indexes are built independently and an issue may only be impacting one or very few tenants. `tempodb_blocklist_tenant_index_builder`,
`tempodb_blocklist_tenant_index_age_seconds` and `tempodb_blocklist_tenant_index_errors_total` are all per-tenant metrics. If
you can isolate the impacted tenants, attempt to take targeted action instead of making sweeping changes. Your easiest lever
to pull is to simply delete stale tenant indexes as all components will fallback to bucket listing. The tenant index is located at:

```
/<tenant>/index.json.gz
```


### Quick checks
- Metric/query: `max by (tenant) (tempodb_blocklist_tenant_index_age_seconds)`
- Metric/query: `sum by (tenant) (tempodb_blocklist_tenant_index_builder)`
- Log query: `{container=~"backend-worker|backend-scheduler|querier|query-frontend"} |= "tenant index"`

## TempoBadOverrides

Fix the overrides. Overrides are loaded by the distributors, so there should hopefully be meaningful logging there.


### Quick checks
- Metric/query: `tempo_runtime_config_last_reload_successful == 0`
- Metric/query: `increase(kube_pod_container_status_restarts_total{container="distributor"}[10m])`
- Log query: `{container="distributor"} |= "override"`

## TempoUserConfigurableOverridesReloadFailing

This alert fires when Tempo repeatedly fails to reload user-configurable overrides from the backend. Per-tenant overrides can become stale until backend access is restored or invalid data is corrected.

How to investigate:

1. Check which component is failing to refresh overrides.
2. Review the backend used to store user-configurable overrides for availability or permission issues.
3. Look for malformed or partially written override payloads for the affected tenants.
4. Confirm whether recent override changes correlate with the failures.


### Quick checks
- Metric/query: `sum by (cluster, namespace) (increase(tempo_overrides_user_configurable_overrides_reload_failed_total[1h]))`
- Metric/query: `sum by (cluster, namespace) (increase(tempo_overrides_user_configurable_overrides_reload_failed_total[5m]))`
- Log query: `{container=~"distributor|querier|query-frontend|metrics-generator"} |= "failed to refresh user-configurable config"`

## TempoCompactionTooManyOutstandingBlocks

This alert fires when there are too many blocks to be compacted for a long period of time.
The alert does not require immediate action, but is a symptom that compaction is under-scaled
and could affect the read path in particular.

In the current backend-scheduler/backend-worker flow, the scheduler exposes `tempodb_compaction_outstanding_blocks` and related backlog metrics, while backend-workers execute the compaction jobs. Use scheduler metrics to quantify the backlog and worker count/resources to decide whether more worker capacity is needed.

How to fix:

Compaction's bottleneck is most commonly CPU time, so adding more backend workers is the most effective measure.

After compaction has been scaled out, it will take time for backend workers to catch
up with their outstanding blocks.
Take a look at `tempodb_compaction_outstanding_blocks` and check if blocks start
going down. If not, further scaling may be necessary.

Since the number of blocks is elevated, it may also be necessary to review the queue-related
settings to prevent [trace lookup failures](#trace-lookup-failures).


### Quick checks
- Metric/query: `sum by (tenant) (tempodb_compaction_outstanding_blocks{container="backend-scheduler"})`
- Metric/query: `count(tempo_build_info{container="backend-worker"})`
- Metric/query: `tempodb_blocklist_length{container="backend-scheduler"}`
- Log query: `{container=~"backend-worker|backend-scheduler"} |= "compaction"`

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
   - For the metrics-generators scale up the consumer group by adding more instances
   - Scale up the block-builder instances to match the live-store partitions
   - For the block-builder consider reducing the cycle time
   - Increase resources (CPU/memory) for the consumer instances
   - Check for and fix any bottlenecks in the processing pipeline
   - If the lag is temporary due to a spike in traffic, monitor to see if it recovers


### Quick checks
- Metric/query: `max by (container, partition, group) (tempo_ingest_group_partition_lag_seconds{container=~"metrics-generator|block-builder|live-store"})`
- Metric/query: `tempo_block_builder_owned_partitions or tempo_live_store_partition_owned`
- Metric/query: `increase(kube_pod_container_status_restarts_total{container=~"metrics-generator|block-builder|live-store"}[10m])`
- Log query: `{container=~"metrics-generator|block-builder|live-store"}`

## TempoLiveStoreSingleMemberLagHigh

This alert fires when a single owner of a live-store partition has been lagging for an extended period. At this threshold, normal replay-on-startup cannot explain the lag — a single member is genuinely stuck. Other zones can still serve reads for the partition, so this is an operational concern rather than an immediate read outage.

See [TempoPartitionLag](#TempoPartitionLag) for general Kafka lag troubleshooting.

### How to investigate

1. Identify which pod owns the lagging partition and whether it has recently restarted.

2. **If the pod recently restarted**, it may be replaying. If the node is over-scheduled with many live-store pods, replay can be abnormally slow due to resource contention. As each pod finishes replaying it frees resources for the remaining ones — this is partially self-healing. To speed recovery, delete a few pods so they reschedule onto less-loaded nodes.

3. **If the pod has not restarted recently**, it is genuinely stuck. Check for Kafka issues specific to one broker or partition (e.g. a single-partition leader re-election).

### How to fix

- **Over-scheduled node:** wait for self-healing, or delete a few lagging pods to spread them across less-loaded nodes.
- **Stuck pod:** restart it.
- **Kafka partition issue:** check the partition ring for skew and consider rebalancing.

## TempoLiveStoreAllMembersLagging

This alert fires when **all** owners of a live-store partition are lagging simultaneously. Unlike [TempoLiveStoreSingleMemberLagHigh](#TempoLiveStoreSingleMemberLagHigh) — where other zones can still serve reads — this means no zone can serve the affected partition. **This is a partial read outage.**

A genuine issue looks like one or more specific partitions showing constant, sustained lag while others recover. Transient spikes across all partitions during a rollout are normal and should clear; a single partition staying flat is the signal that something is wrong.

See [TempoPartitionLag](#TempoPartitionLag) for general Kafka lag troubleshooting.

### How to investigate

1. Check whether all live-store pods restarted around the same time (coordinated rollout, node failure, or zone-wide issue).

2. If pods restarted together, check whether any nodes are over-scheduled — if multiple pods land on the same node, simultaneous replay can exhaust node resources and cause all of them to lag together. This is partially self-healing as pods finish replaying and free resources.

3. Check for Kafka broker issues — a partition leader election or broker restart can cause all consumers in a group to lag briefly before recovering.

### How to fix

- **Coordinated restart / node over-scheduling:** wait for self-healing, or delete a few pods to spread them across less-loaded nodes.
- **Kafka root cause:** investigate broker health and partition leadership.
- **Persistent stall:** restart affected pods and monitor lag recovery.

## TempoBlockBuildersPartitionsMismatch

This alert fires when more than one active or inactive partition has been unowned by a block-builder for more than 10 minutes.

### How it works

Each block-builder is configured to consume from one or more Kafka partitions.

The distributor uses the partition ring to determine which partitions are active and where each trace should be sent.

### How to investigate it

1. Ensure that all the block-builders are running and not crashlooping.
2. Look at which partition is missing. This metric will return the assigned ones and their state:
   `tempo_block_builder_owned_partitions{namespace="tempo-dev-01"}`
3. Check whether that partition is assigned to any of the block-builder instances. Look for any error.


### Quick checks
- Metric/query: `max by (cluster, namespace) (tempo_partition_ring_partitions{name="livestore-partitions", state=~"Active|Inactive"})`
- Metric/query: `sum by (cluster, namespace) (tempo_block_builder_owned_partitions)`
- Log query: `{container="block-builder"}`

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

3. Look for errors in the logs related to live-store pod health.

### How to fix

1. **If live-store pods are down or crashlooping:**
   - Check pod status and logs for errors
   - If OOM, consider scaling resources
   - Restart unhealthy pods if needed

2. **If live-store pods are running but not claiming partitions:**
   - Check logs for errors


### Quick checks
- Metric/query: `tempo_partition_ring_partitions{name="livestore-partitions", state=~"Active|Inactive"}`
- Metric/query: `tempo_live_store_partition_owned{}`
- Metric/query: `increase(kube_pod_container_status_restarts_total{container="live-store"}[10m])`
- Log query: `{container="live-store"}`

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
- Verify the zone has sufficient resources to run live-store pods


### Quick checks
- Metric/query: `count by (zone) (tempo_live_store_partition_owned)`
- Metric/query: `increase(kube_pod_container_status_restarts_total{container="live-store"}[10m])`
- Log query: `{container="live-store"}`

## TempoDistributorUsageTrackerErrors

The Tempo distributor is encountering errors when doing usage tracking/cost attribution for a tenant. This is usually caused by a misconfigured usage tracker configuration for the affected tenant.

This alert means that the tenant's cost attribution is not working or is incorrect due to this misconfiguration.

### Troubleshooting Steps

1. Check the alert labels to identify the affected tenant (the alerting metric has the tenant, reason, cluster, and namespace labels).
2. Review the tenant's usage tracker configuration for issues.
3. Check distributor logs for detailed error messages and more details about the reason.
4. If the configuration is incorrect, work with the impacted tenant to fix their usage tracker settings.


### Quick checks
- Metric/query: `sum by (tenant, reason) (rate(tempo_distributor_usage_tracker_errors_total[5m]))`
- Metric/query: `sum by (tenant, reason) (increase(tempo_distributor_usage_tracker_errors_total[1h]))`
- Log query: `{container="distributor"} |= "failed to collect usage tracker metric"`

## TempoMetricsGeneratorProcessorUpdatesFailing

The metrics-generator contains processors to convert spans into metrics. They can be enabled or disabled dynamically per
tenant. The metrics-generator periodically reads the overrides file and updates the active processors accordingly.

Updating the processors might fail if an invalid processor is configured (for example a typo) or if creating and registering
the new processor failed. If the update failed, the metrics-generator will keep retrying. This process is per tenant and
failed updates to a single tenant should not impact others.

How to investigate:

1. Check the logs.
2. Verify if there were any recent changes to the overrides file.


### Quick checks
- Metric/query: `sum by (tenant) (increase(tempo_metrics_generator_active_processors_update_failed_total[5m]))`
- Metric/query: `sum by (tenant, processor) (tempo_metrics_generator_active_processors)`
- Log query: `{container="metrics-generator"} |= "updating the processors failed"`

## TempoMetricsGeneratorServiceGraphsDroppingSpans

The service graphs processor buffers spans internally. If the buffer is full, spans will be dropped. This will result in
incomplete service graphs metrics.

How to fix:

1. Increase buffer size or increase the amount of workers.


### Quick checks
- Metric/query: `sum by (tenant) (increase(tempo_metrics_generator_processor_service_graphs_dropped_spans[1h]))`
- Metric/query: `sum by (tenant) (increase(tempo_metrics_generator_spans_received_total[1h]))`
- Log query: `{container="metrics-generator"} |= "service graph"`

## TempoMetricsGeneratorCollectionsFailing

The metrics-generator has a registry that keeps track of all the active counters. On a regular interval the state of
these counters will be collected and appended as samples onto the WAL. These samples are sent out using remote write.

If a collection fails, the samples will be dropped and the metrics for this tenant will be interrupted.

How to investigate:

1. Check the logs to find out why the collection failed. Search for the phrase "collecting metrics failed".


### Quick checks
- Metric/query: `sum by (tenant, pod) (increase(tempo_metrics_generator_registry_collections_failed_total[5m]))`
- Metric/query: `max by (pod) (container_memory_working_set_bytes{container="metrics-generator"})`
- Log query: `{container="metrics-generator"} |= "collecting metrics failed"`

## TempoMemcachedErrorsElevated

This alert fires when memcached request errors exceed 20% for a role in a namespace/cluster.

Likely causes include an overloaded memcached tier, network issues, or backend store timeouts surfacing as 5xx errors.

### How to investigate

1. Check memcached pod health and recent restarts.
2. Review memcached logs for 5xx responses or connection errors.
3. Compare request rate and cache hit ratio to see if the tier is saturated.
4. Look for backend store or network timeouts that could be causing request failures.

### How to fix

- Scale memcached for the affected role (more replicas or resources)
- Increase memcached capacity limits if eviction or memory pressure is high
- Address backend or network issues if errors correlate with timeouts upstream


### Quick checks
- Metric/query: `sum(rate(tempo_memcache_request_duration_seconds_count{status_code="500"}[5m])) by (cluster, namespace, name)`
- Metric/query: `sum(rate(tempo_memcache_request_duration_seconds_count{}[5m])) by (cluster, namespace, name, status_code)`
- Log query: `{container=~"querier|memcached"} |= "memcache"`

## TempoBackendSchedulerJobsFailureRateHigh

This alert fires when the failure rate of backend scheduler jobs exceeds the alert threshold.

This is a strong signal that job processing is having issues and jobs are failing to schedule.

- Inspect logs of the backend scheduler component for errors
- Identify job types and tenants contributing to failures

If the failure rate stays high, the blocklist might grow because compaction might not happen due to failed job scheduling.


### Quick checks
- Metric/query: `sum(increase(tempo_backend_scheduler_jobs_failed_total[5m])) by (cluster, namespace) / sum(increase(tempo_backend_scheduler_jobs_created_total[5m])) by (cluster, namespace)`
- Metric/query: `sum(tempo_backend_scheduler_jobs_active) by (cluster, namespace)`
- Log query: `{container="backend-scheduler"}`

## TempoBackendSchedulerRetryRateHigh

This alert fires when a high number of jobs are being retried by workers,
indicating execution instability or transient failures or possible issues with backend object stores.

- Check logs for tenants and job types that are being retried
- Look at logs to see if the backend object store is having issues or not


### Quick checks
- Metric/query: `sum(increase(tempo_backend_scheduler_jobs_retry_total[1m])) by (cluster, namespace)`
- Metric/query: `sum(increase(tempo_backend_scheduler_jobs_failed_total[5m])) by (cluster, namespace)`
- Log query: `{container="backend-scheduler"}`

## TempoBackendSchedulerCompactionEmptyJobRateHigh

This alert fires when a high number of jobs received by the backend scheduler
are empty. This can happen when the backoff for the compaction provider is too
low and the outstanding blocklist is low, meaning that there is not enough work
to do.

- Check logs for the backend scheduler component for errors
- Check the blocklist and compare it to the outstanding block list


### Quick checks
- Metric/query: `sum(increase(tempo_backend_scheduler_compaction_tenant_empty_job_total[1m])) by (cluster, namespace)`
- Metric/query: `sum by (tenant) (tempodb_compaction_outstanding_blocks{container="backend-scheduler"})`
- Log query: `{container="backend-scheduler"} |= "compaction"`

## TempoBackendWorkerBadJobsRateHigh

This alert fires when backend workers receive an elevated number of bad jobs.

This should not fire under normal operations and is most likely due to bad jobs being generated by the scheduler.

This can also happen due to a bug or version/schema mismatch where a new version of scheduler is generating
jobs while the worker is on an old version/schema.

Audit the jobs that are marked as bad and fix them to resolve this alert.


### Quick checks
- Metric/query: `sum(increase(tempo_backend_worker_bad_jobs_received_total[1m])) by (cluster, namespace)`
- Metric/query: `sum(increase(tempo_backend_scheduler_jobs_created_total[5m])) by (cluster, namespace)`
- Log query: `{container=~"backend-worker|backend-scheduler"} |= "job"`

## TempoBackendWorkerCallRetriesHigh

This alert fires when backend workers retry calls frequently to the scheduler.

This can mean multiple things:
- Scheduler is down or unreachable from workers
- Scheduler is rate limiting the workers
- Scheduler is having issues handing out jobs or failing internally due to outage/issues on the backend object store

Look at the logs for scheduler and worker and fix the issues.


### Quick checks
- Metric/query: `sum(increase(tempo_backend_worker_call_retries_total[1m])) by (cluster, namespace)`
- Metric/query: `increase(kube_pod_container_status_restarts_total{container=~"backend-worker|backend-scheduler"}[10m])`
- Log query: `{container="backend-worker"} |= "error calling scheduler"`

## TempoVultureHighErrorRate

This alert fires when Tempo vulture detects a high error rate (above the configured threshold) while validating write or read paths. It indicates there are problems with trace processing or storage.


### Quick checks
- Metric/query: `sum(rate(tempo_vulture_trace_error_total[1m])) by (cluster, namespace, error)`
- Metric/query: `sum(rate(tempo_vulture_trace_total[1m])) by (cluster, namespace)`
- Log query: `{container=~"tempo-vulture|vulture"}`
