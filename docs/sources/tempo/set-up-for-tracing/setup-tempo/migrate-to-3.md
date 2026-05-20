---
title: Migrate from Tempo 2.x to 3.0
menuTitle: Migrate to 3.0
description: Migrate a self-managed Grafana Tempo deployment from the 2.x ingester-based architecture to the 3.0 Kafka-based architecture.
weight: 520
aliases:
  - ../migrate-to-3/
topicType: task
versionDate: 2026-02-25
---

# Migrate from Tempo 2.x to 3.0

Grafana Tempo 3.0 introduces a new architecture that replaces ingesters with a Kafka-based ingest path.
Distributors write trace data to Kafka, and two new components consume from it: block-builders create blocks for long-term object storage, and live-stores serve recent-data queries. A backend-scheduler and backend-worker replace the compactor for block maintenance.

This guide walks you through migrating a self-managed Grafana Tempo deployment from 2.x to 3.0.
The migration path depends on your deployment mode:

- **Monolithic mode** users have a simpler path: update the configuration and upgrade the binary. See [Migrate a monolithic deployment](#migrate-a-monolithic-deployment).
- **Microservices mode** users follow a parallel-deployment migration: deploy 3.0 alongside 2.x, switch traffic, then decommission. The bulk of this guide covers that path.

{{< admonition type="warning" >}}
There's no in-place downgrade from 3.0 to 2.x. During the migration you can route traffic back to 2.x (see [Roll back](#roll-back)), but once you decommission the 2.x deployment, plan to stay on 3.0.
{{< /admonition >}}

{{< admonition type="note" >}}
Running two Tempo deployments in parallel increases infrastructure costs for the duration of the migration. Plan to complete the migration and decommission the 2.x deployment promptly.
{{< /admonition >}}

## Before you begin

Confirm the following before you start:

- Your Tempo 2.x deployment uses **vParquet4 or later** as the block format. Tempo 3.0 doesn't support vParquet3 or earlier. If you're using an older format, upgrade your block format before migrating. Refer to [Choose a different block format](/docs/tempo/<TEMPO_VERSION>/configuration/parquet/#choose-a-different-block-format).
- **Microservices mode only**: You have a running **Kafka-compatible system** (for example, Apache Kafka or Redpanda). Monolithic mode does not require Kafka.
- You have access to the **same object storage** bucket or container used by your 2.x deployment.
- If you're running **scalable monolithic mode** (SSB), plan to switch to either monolithic or microservices mode. SSB has been removed in Tempo 3.0.
- You've reviewed the [Upgrade your Tempo installation](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/upgrade/) page for breaking changes.

### Migrate legacy overrides format

Tempo 3.0 disables the legacy (flat, unscoped) overrides format by default. If your main config or per-tenant overrides file uses the legacy format, Tempo refuses to start with an error like:

```
DEPRECATED: legacy overrides config format detected but legacy overrides are disabled by default.
Migrate your overrides config to the new scoped format, or set -config.enable-legacy-overrides=true
(or enable_legacy_overrides: true in YAML) to continue using legacy overrides temporarily
```

As a temporary workaround, set `enable_legacy_overrides: true` in the `overrides` block or pass `-config.enable-legacy-overrides=true` on the CLI. Legacy overrides will be removed in a future release.

To migrate, use the [`tempo-cli migrate overrides-config`](/docs/tempo/<TEMPO_VERSION>/operations/tempo_cli/#migrate-overrides-config-command) command, or manually rewrite your overrides to the new scoped format. Refer to the [overrides configuration reference](/docs/tempo/<TEMPO_VERSION>/configuration/#overrides) for the full schema.

## Architecture changes

In Tempo 2.x, distributors forward spans to ingesters and metrics-generators over gRPC. Ingesters batch spans, build blocks, and flush them to object storage. Compactors maintain blocks in storage. For more information, refer to the [Tempo 2.x architecture](https://grafana.com/docs/tempo/v2.10.x/introduction/architecture/).

In Tempo 3.0, distributors write spans to Kafka instead. **Block-builders** consume from Kafka and build blocks for object storage. **Live-stores** consume from Kafka and serve recent-data queries (typically the last 30 minutes to 1 hour). A **backend-scheduler** and **backend-worker** replace the compactor for block compaction and retention. Because Kafka provides durability, Tempo 3.0 operates with a replication factor of 1 (RF1), eliminating the need for ingester replication. Ingesters, the scalable single binary (SSB) mode, and the compactor target are removed.

For a detailed description of the new architecture, refer to [Tempo architecture](/docs/tempo/<TEMPO_VERSION>/introduction/architecture/).

## Migrate a monolithic deployment

Monolithic deployments don't require Kafka or any new components beyond the binary upgrade. The distributor pushes spans directly to the in-process metrics-generator (the same as in 2.x), and the live-store handles both serving recent queries and flushing blocks to storage.

To migrate a monolithic deployment:

1. Migrate your configuration. The simplest way is to use the [`tempo-cli migrate config`](/docs/tempo/<TEMPO_VERSION>/operations/tempo_cli/#migrate-config-command) command:

   ```bash
   tempo-cli migrate config --mode=monolithic old-config.yaml > new-config.yaml
   ```

   Or update the config manually: remove the `ingester:`, `ingester_client:`, `compactor:`, and `metrics_generator_client:` blocks. If your metrics-generator uses the `local_blocks` processor, remove it (block building is handled internally by the live-store). All other blocks carry over unchanged.

1. Upgrade the binary to Tempo 3.0 with the new configuration.

1. Validate the deployment with [Tempo Vulture](/docs/tempo/<TEMPO_VERSION>/operations/tempo-vulture/), or by querying historical traces directly to confirm the new deployment can read from your object storage.

Tempo 3.0 reads both 2.x blocks (RF3) and 3.0 blocks (RF1) automatically, so historical queries work without configuration. TraceQL metrics queries only read RF1 blocks — if your 2.x deployment didn't use the `local-blocks` processor, metrics queries only return results for data ingested after the upgrade.

The rest of this guide covers microservices mode.

## Prepare for migration

Before deploying Tempo 3.0, configure a Kafka topic and prepare your new configuration.

### Configure the Kafka topic

Tempo uses a single Kafka topic for all trace data, configured through `ingest.kafka.topic`.
You can let Tempo auto-create the topic or create it manually.
For more information, refer to the [Kafka topic configuration reference](https://kafka.apache.org/documentation/#topicconfigs).

To let Tempo create the topic automatically:

```yaml
ingest:
  kafka:
    address: <KAFKA_BROKER_ADDRESS>
    topic: <KAFKA_TOPIC_NAME>
    auto_create_topic_enabled: true
    auto_create_topic_default_partitions: <PARTITION_COUNT>
```

Your Kafka broker must also have `auto.create.topics.enable` set to `true`.

{{< admonition type="warning" >}}
`auto_create_topic_default_partitions` sets the broker-wide `num.partitions` default, which affects all auto-created topics on the broker, not just the Tempo topic. If other services share the same Kafka cluster, consider creating the topic manually instead.
{{< /admonition >}}

To create the topic manually, set the partition count based on your expected parallelism. As a starting point, plan for approximately 10 MB/s of peak ingestion throughput per partition.

### Review configuration changes

Tempo 3.0 removes ingester configuration and adds new configuration blocks. The simplest way to migrate your config is to use the [`tempo-cli migrate config`](/docs/tempo/<TEMPO_VERSION>/operations/tempo_cli/#migrate-config-command) command, which takes your 2.x config and outputs a valid 3.0 config:

```bash
tempo-cli migrate config --kafka-address=<KAFKA_BROKER_ADDRESS> --kafka-topic=<KAFKA_TOPIC> old-config.yaml > new-config.yaml
```

The command removes ingester, `ingester_client`, compactor, and `metrics_generator_client` blocks; adds the `ingest:` block in microservices mode; removes `local_blocks` processor config; and sets `compaction_disabled: true` in the defaults overrides for parallel operation. Review the output before deploying.

If you prefer to migrate your config manually:

**Remove** from your 2.x configuration:

- The `ingester:` block and all its settings
- Any `ingester_client:` configuration
- The `compactor:` block (replaced by `backend_scheduler:` and `backend_worker:`)
- Any `metrics_generator_client:` configuration
- Any `local_blocks` processor configuration

**Add** to your 3.0 configuration:

- `ingest:` block to connect Tempo to Kafka

Your existing `server:`, `distributor:`, `query_frontend:`, `storage:`, `memberlist:`, and `overrides:` blocks carry over largely unchanged.

The following example shows the minimum new configuration to add for microservices mode:

```yaml
ingest:
  kafka:
    address: <KAFKA_BROKER_ADDRESS>
    topic: <KAFKA_TOPIC>
```

Each block-builder instance consumes from exactly one Kafka partition based on its ordinal: block-builder-0 consumes partition 0, block-builder-1 consumes partition 1, and so on. This means the number of block-builder replicas must equal the number of Kafka partitions.

To keep block-builder replicas in sync with the partition count, scale them to match the live-store replica count (live-stores also run one replica per partition). You can do this with:

- A **KEDA autoscaler** that mirrors the live-store pod count. The Tempo Jsonnet library includes this configuration.
- The [**Grafana rollout-operator**](https://github.com/grafana/rollout-operator) mirroring feature, which keeps one 
  StatefulSet's replica count in sync with another.
- A **static replica count** set to the number of Kafka partitions if your partition count is fixed.

The `live_store:` block uses sensible defaults and doesn't require overrides for most deployments.

If your Kafka cluster requires authentication, add SASL credentials to the `ingest` block:

```yaml
ingest:
  kafka:
    address: <KAFKA_BROKER_ADDRESS>
    topic: <KAFKA_TOPIC>
    sasl_username: <USERNAME>
    sasl_password: <PASSWORD>
```

For the full list of options, refer to the [Ingest](/docs/tempo/<TEMPO_VERSION>/configuration/#ingest), [Block-builder](/docs/tempo/<TEMPO_VERSION>/configuration/#block-builder), and [Live-store](/docs/tempo/<TEMPO_VERSION>/configuration/#live-store) configuration reference.

#### Disable compaction during parallel operation

Only one compaction system can safely write to shared storage at a time. Disable compaction in the 3.0 deployment while the 2.x compactors are still running.

Set `compaction_disabled: true` in the defaults and in every per-tenant override in your 3.0 configuration. Overrides don't inherit properties, so each tenant entry must include it explicitly:

```yaml
overrides:
  defaults:
    compaction:
      compaction_disabled: true
  "tenant-a":
    compaction:
      compaction_disabled: true
  # Repeat for all per-tenant overrides
```

You re-enable compaction by removing this override after the 2.x deployment is decommissioned. Refer to [Clean up the Tempo 2.x deployment](#clean-up-the-tempo-2x-deployment).

#### Querying historical data

Tempo 2.x ingesters wrote blocks with replication factor 3 (RF3). Tempo 3.0 block-builders write blocks with replication factor 1 (RF1). After migration, your object storage contains both types. Tempo 3.0 automatically queries the correct blocks based on their replication factor — no additional configuration is needed.

TraceQL and trace-by-ID queries work across your full history, covering both old RF3 blocks from 2.x and new RF1 blocks from 3.0.

TraceQL metrics queries only read RF1 blocks. If your 2.x deployment used the `local-blocks` processor in the metrics-generator, those RF1 blocks are already in storage and metrics queries cover your full history. If you didn't use the `local-blocks` processor, metrics queries only return results for data ingested after the switch to 3.0.

#### Metrics-generator

In Tempo 3.0, the metrics-generator consumes spans from Kafka in microservices mode. It automatically uses the top-level `ingest:` block — no additional Kafka configuration is needed for it.

The `local-blocks` processor has been removed — block building is now handled by the block-builder component. Remove any `local_blocks` configuration from your metrics-generator settings.

For more information, refer to [Metrics-generator](/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/metrics-generator/).

## Deploy Tempo 3.0

Deploy a new Tempo 3.0 instance alongside your existing 2.x deployment. Both deployments must point at the same object storage bucket so the new deployment can query historical blocks.

For deployment instructions, refer to [Deploy Tempo](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/deploy/). For a complete example microservices configuration, refer to the [distributed docker-compose example](https://github.com/grafana/tempo/tree/main/example/docker-compose/distributed).

To validate the deployment:

1. Deploy Tempo 3.0 using your preferred method with the updated configuration.

1. Confirm that compaction is disabled in the 3.0 deployment, as described in [Disable compaction during parallel operation](#disable-compaction-during-parallel-operation).

1. Verify that all components start successfully and connect to Kafka. Check that `tempo_live_store_ready` equals `1`. The log message `"live-store ready to serve queries"` confirms the live-store is ready. At this point, no traffic is flowing through the 3.0 distributors yet, so write metrics will be at zero.

   {{< admonition type="note" >}}
   On a fresh deployment with no traffic flowing yet, the live-store can take up to `live_store.readiness_max_wait` (default 30 minutes) to become ready, because there's no Kafka high-water mark to compare against. This is normal. Once traffic starts flowing in the next step, restarts are near-instant.
   {{< /admonition >}}

1. Validate the deployment end-to-end using [Tempo Vulture](/docs/tempo/<TEMPO_VERSION>/operations/tempo-vulture/), Tempo's built-in testing tool. Vulture writes traces to the distributor and reads them back through the query frontend, validating the full write and read path. Point it at the 3.0 deployment and let it run for 10–15 minutes. Confirm that `tempo_vulture_trace_error_total` stays at zero and `tempo_vulture_trace_total` is increasing.

   You can also validate manually by querying the 3.0 deployment directly — for example, using `curl` against the query frontend API or querying from Grafana:

   ```bash
   curl http://<TEMPO_3_QUERY_FRONTEND>:3200/api/traces/<TRACE_ID>
   ```

   Use a trace ID that you know exists in your 2.x deployment. If the query returns trace data, the new deployment is reading from storage correctly.

Before proceeding to the cutover, confirm all of the following:

- `tempo_live_store_ready` equals `1` on every live-store replica.
- `rate(tempo_block_builder_fetch_errors_total[5m])` is at or near zero on every block-builder.
- Vulture has been running for 10–15 minutes with `tempo_vulture_trace_error_total` at zero and `tempo_vulture_trace_total` increasing.
- At least one trace-by-ID query for a trace that exists in your 2.x deployment returns data when sent to the 3.0 query frontend.

Resolve any issues so all of these conditions are satisfied. Switching traffic with an unhealthy 3.0 deployment risks data loss for newly ingested traces.

## Switch traffic to Tempo 3.0

Redirect your trace clients to send data to the Tempo 3.0 deployment. Trace clients include OpenTelemetry Collectors, Grafana Alloy instances, or other exporters configured to send spans to Tempo.

To switch traffic to the 3.0 deployment:

1. If your routing layer supports gradual traffic shifting, start by sending a small percentage of traffic to the 3.0 deployment and monitor for errors.

1. Once canary traffic is stable, update your routing to send all traffic to the 3.0 deployment. This is typically a DNS record change, load balancer update, or collector configuration change.

1. Monitor the 3.0 deployment. Confirm:
   - `tempo_distributor_kafka_write_bytes_total` is increasing steadily.
   - `tempo_block_builder_fetch_records_total` is increasing and `tempo_block_builder_fetch_errors_total` is at zero.
   - `tempo_live_store_ready` equals `1` and `tempo_live_store_records_processed_total` is increasing.
   - Queries return results for newly ingested traces.

1. Confirm traffic has drained from the 2.x deployment. Check the 2.x distributor metrics to verify it's no longer receiving requests.

Before proceeding to cleanup, confirm that the 3.0 deployment is handling all traffic and the 2.x deployment shows zero active requests.

## Clean up the Tempo 2.x deployment

After traffic has fully moved to the 3.0 deployment, decommission the old deployment.

To clean up the 2.x deployment:

1. Wait for the 2.x ingesters to flush all in-memory traces to object storage. With default settings (`max_block_duration` of 30 minutes and `complete_block_timeout` of 15 minutes), this takes up to approximately 45 minutes. If you've customized these values, add your `max_block_duration` and `complete_block_timeout` together to estimate the wait.

   To confirm the ingesters are drained, check these metrics on the 2.x deployment:
   - `tempo_ingester_live_traces` should drop to zero (or near zero) — no traces remain in memory.
   - `tempo_ingester_flush_queue_length` should be zero — no pending flushes.

1. Scale the 2.x compactors to zero and remove the `compaction_disabled` override from the 3.0 deployment to re-enable compaction. Running without compaction for extended periods can cause the blocklist to grow.

1. Once you're confident the 3.0 deployment is stable, scale all 2.x components to zero and decommission the old infrastructure. Don't delete the shared object storage.

1. Review your backend-worker replica count. The overlap period may have produced additional blocks that require compaction. Once the backlog clears, reduce backend-workers to your steady-state configuration.

## Verify the migration

After completing the migration, confirm the following:

- Distributors are writing to Kafka. Verify that `tempo_distributor_kafka_write_bytes_total` is increasing and `tempo_distributor_kafka_write_latency_seconds` is within an acceptable range.
- Block-builders are consuming from Kafka and building blocks. Verify that `tempo_block_builder_fetch_records_total` is increasing and `tempo_block_builder_fetch_errors_total` is at zero.
- Live-stores are serving recent-data queries. Verify that `tempo_live_store_ready` equals `1` and `tempo_live_store_records_processed_total` is increasing. Confirm `tempo_live_store_records_dropped_total` is at zero.
- Kafka lag is within an acceptable range. Check `tempo_ingest_group_partition_lag_seconds`.
- Historical queries return results from object storage.
- Backend-workers are running in the 3.0 deployment and compacting blocks.
- If you use the metrics-generator, it's producing metrics from Kafka-sourced trace data.

## Roll back

Rolling back depends on how far the migration has progressed.

### Before switching traffic

The 2.x deployment is still active and serving all traffic. No rollback is needed. Stop the 3.0 deployment.

### After switching traffic, before decommissioning 2.x

Revert the traffic routing change to point clients back to the 2.x deployment. Because the cutover is a routing change (DNS, load balancer, or collector config), this is near-instant. Both deployments share the same object storage, so historical data remains available. Traces ingested by the 3.0 deployment into Kafka may not be queryable from the 2.x deployment if block-builders haven't flushed them to storage yet.

### After decommissioning 2.x

Rolling back requires redeploying the 2.x infrastructure. Blocks written by the 3.0 block-builders remain in object storage and are readable by both versions, assuming they use a compatible block format (vParquet4 or later).

## Troubleshoot the migration

This section covers common issues you might encounter during the migration.

### Kafka connection failure

Tempo logs errors such as `"the Kafka address has not been configured"`, `"ping kafka; will retry"`, or `"kafka broker not ready after 10 retries"`.

To resolve this issue:

- Verify that `ingest.kafka.address` in your configuration points to the correct broker address.
- Confirm the broker is reachable from the Tempo deployment. Check network connectivity and firewall rules.
- If you use SASL authentication, verify that both `sasl_username` and `sasl_password` are set. Setting only one produces the error `"the SASL username and password must be both configured to enable SASL authentication"`.

### Block-builder not consuming

Tempo logs errors such as `"no partitions assigned"` or `"failed to create kafka reader client"`.

To resolve this issue:

- Verify the Kafka topic exists and contains data.
- Check the `tempo_block_builder_owned_partitions` metric to confirm the block-builder has partition assignments.
- Verify the block-builder can reach the Kafka broker.

### Live-store not ready

Queries return errors with `"live-store is starting"` or the live-store logs show `"failed to catch up"`.

The live-store needs time to catch up with Kafka after startup. To monitor progress:

- Check the `tempo_live_store_ready` metric. A value of `1` indicates the live-store is ready to serve queries.
- Check `tempo_live_store_catch_up_duration_seconds` to monitor catch-up progress.
- The log message `"live-store ready to serve queries"` confirms readiness.

### Historical queries return no results

Queries for traces that existed in your 2.x deployment return empty results from the 3.0 deployment.

To resolve this issue:

- Verify that the 3.0 deployment uses the same object storage configuration (bucket, endpoint, credentials) as the 2.x deployment.
- Confirm that the storage backend is reachable from the 3.0 deployment.

## Next steps

- Refer to [Upgrade your Tempo installation](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/upgrade/) for version-specific breaking changes and removed configuration parameters.
- Refer to [Tempo architecture](/docs/tempo/<TEMPO_VERSION>/introduction/architecture/) for details on how block-builders, live-stores, and Kafka interact.
- Refer to [Configuration](/docs/tempo/<TEMPO_VERSION>/configuration/) for the full configuration reference, including all `ingest:`, `block_builder:`, and `live_store:` parameters.
- Refer to [Plan your deployment](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/plan/) for guidance on deployment modes and sizing in Tempo 3.0.
