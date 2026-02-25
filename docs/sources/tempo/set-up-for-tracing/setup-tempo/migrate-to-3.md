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

Grafana Tempo 3.0 introduces a new architecture that replaces ingesters with an Apache Kafka-based ingest path. Distributors write trace data to Kafka, and two new components consume from it. Block-builders create blocks for long-term object storage, and live-stores serve recent-data queries.
Refer to [the Kafka Introduction](https://kafka.apache.org/42/getting-started/introduction/) for more information about Kafka.

This guide walks you through migrating a self-managed Grafana Tempo deployment from 2.x to 3.0.
Because the architecture change is fundamental, this is a migration rather than an in-place upgrade.
You deploy a new Tempo 3.0 instance alongside your existing 2.x deployment, switch traffic, and decommission the old deployment.

{{< admonition type="warning" >}}
There's no downgrade path from 3.0 to 2.x. Once you begin writing blocks with Tempo 3.0, you can't revert to a 2.x deployment.
{{< /admonition >}}

The active migration steps take approximately 1-2 hours to complete, followed by a one-week validation period before final decommission.

{{< admonition type="note" >}}
Running two Tempo deployments in parallel increases infrastructure costs for the duration of the migration. Plan to complete the migration and decommission the 2.x deployment promptly.
{{< /admonition >}}

## Before you begin

Confirm the following before you start:

- Your Tempo 2.x deployment uses vParquet4 or later as the block format. Tempo 3.0 doesn't support vParquet3 or earlier. If you're using an older format, upgrade your block format before migrating. Refer to [Choose a different block format](/docs/tempo/<TEMPO_VERSION>/configuration/parquet/#choose-a-different-block-format).
- You have a running Kafka-compatible system, for example, Apache Kafka, Redpanda, or WarpStream. Tempo 3.0 requires Kafka for all deployment modes, including monolithic. Refer to your Kafka provider's documentation for installation instructions.
- Your Kafka cluster is sized to handle your trace ingestion throughput. As a starting point, match the number of Kafka partitions to your expected parallelism for block-builders and live-stores.
- You have access to the same object storage bucket or container used by your 2.x deployment.
- You control the traffic routing layer (load balancer, DNS, or reverse proxy) that directs trace clients to Tempo.
- You have your Tempo 2.x configuration file and any per-tenant overrides available.
- If you're running scalable monolithic mode (SSB), plan to switch to either monolithic or microservices mode. SSB has been removed in Tempo 3.0.
- You've reviewed the [Upgrade your Tempo installation](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/upgrade/) page for breaking changes and removed configuration parameters.

## Review the architecture changes

Tempo 2.x and 3.0 handle trace ingestion differently. Understanding these changes helps you plan your configuration and deployment.

In Tempo 2.x, distributors forward spans directly to ingesters over gRPC. Ingesters batch spans in memory, build blocks, and flush them to object storage. Queriers read both from ingesters (for recent data) and from object storage (for historical data).
For more information about the Tempo 2.x architecture, refer to [Tempo 2.x architecture](https://grafana.com/docs/tempo/v2.10.x/introduction/architecture/).

In Tempo 3.0, distributors write spans to Kafka. Two components consume from Kafka independently:

- **Block-builders** consume spans and build blocks for long-term object storage.
- **Live-stores** consume spans and serve recent-data queries, typically covering the last 30 minutes to 1 hour.

Queriers read from live-stores for recent data and from object storage for historical data.

Because Kafka provides durability, Tempo 3.0 operates with a replication factor of 1 (RF1). Data is durable as soon as Kafka acknowledges the write. This eliminates the need for ingester replication and deduplication.

For a detailed description of each component, refer to [Tempo architecture](/docs/tempo/<TEMPO_VERSION>/introduction/architecture/).

## Prepare for migration

Before deploying Tempo 3.0, create the Kafka topic and review the configuration changes you need to make.

A Kafka topic is a named channel that organizes messages within a Kafka cluster. Producers write
messages to a topic, and consumers read messages from it.
A topic is divided into partitions, which are ordered, append-only logs. Partitions allow multiple
consumers to read from the same topic in parallel, where each partition is consumed by a different
instance.

### Configure the Kafka topic

Tempo uses a single Kafka topic for all trace data, configured through `ingest.kafka.topic`.
You can let Tempo auto-create the topic or create it manually with settings that match your throughput.
For more information about Kafka topics, refer to the [Kafka topic configuration reference](https://kafka.apache.org/documentation/#topicconfigs).

To let Tempo create the topic automatically, enable auto-creation in the `ingest` block.

```yaml
ingest:
  enabled: true
  kafka:
    address: <KAFKA_BROKER_ADDRESS>
    topic: <KAFKA_TOPIC_NAME>
    auto_create_topic_enabled: true
    auto_create_topic_default_partitions: 1000
```

Your Kafka broker must also have `auto.create.topics.enable` set to `true`.

To create the topic manually, set the partition count based on your expected parallelism. Each partition supports approximately one block-builder or live-store instance. For most deployments, 1000 partitions provides sufficient headroom. For sizing guidance, refer to [Size your cluster](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/plan/size/).

### Review configuration changes

Tempo 3.0 removes ingester configuration and adds new configuration blocks. At a minimum, you need to make the following changes.

**Remove** from your 2.x configuration:

- The `ingester:` block and all its settings
- Any ingester ring configuration

**Add** to your 3.0 configuration:

The `ingest:` block to connect Tempo to Kafka:

```yaml
ingest:
  enabled: true
  kafka:
    address: <KAFKA_BROKER_ADDRESS>
    topic: <KAFKA_TOPIC>
```

The `block_builder:` block to configure how blocks are built:

```yaml
block_builder:
  consume_cycle_duration: 5m
```

The `live_store:` block to configure recent-data serving:

```yaml
live_store:
  max_block_duration: 30m
  complete_block_timeout: 1h
```

If your Kafka cluster requires authentication, add the Simple Authentication and Security Layer (SASL) credentials:

```yaml
ingest:
  enabled: true
  kafka:
    address: <KAFKA_BROKER_ADDRESS>
    topic: <KAFKA_TOPIC>
    sasl_username: <USERNAME>
    sasl_password: <PASSWORD>
```

Your existing `server:`, `distributor:`, `querier:`, `query_frontend:`, `compactor:`, `storage:`, `memberlist:`, and `overrides:` blocks carry over unchanged. Copy them from your 2.x configuration into your 3.0 configuration file.

If you use the metrics-generator, note that it also consumes from Kafka in Tempo 3.0. Ensure the metrics-generator configuration includes the same `ingest:` block so it can connect to your Kafka cluster.

For the full list of configuration parameters, refer to the [Ingest](/docs/tempo/<TEMPO_VERSION>/configuration/#ingest), [Block-builder](/docs/tempo/<TEMPO_VERSION>/configuration/#block-builder), and [Live-store](/docs/tempo/<TEMPO_VERSION>/configuration/#live-store) configuration reference.

## Deploy Tempo 3.0

Deploy a new Tempo 3.0 instance alongside your existing 2.x deployment. Both deployments use the same object storage.

To deploy and validate Tempo 3.0:

1. Deploy Tempo 3.0 using your preferred method. Use the updated configuration with the `ingest:`, `block_builder:`, and `live_store:` blocks.

   For Helm (monolithic):

   ```bash
   helm upgrade --install tempo grafana/tempo -f tempo-values.yaml
   ```

   For Helm (microservices):

   ```bash
   helm upgrade --install tempo grafana/tempo-distributed -f tempo-values.yaml
   ```

   For Docker Compose:

   ```bash
   docker compose up -d
   ```

   For detailed deployment instructions, refer to [Deploy with Helm](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/deploy/kubernetes/helm-chart/) or [Deploy with Docker Compose](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/deploy/locally/docker-compose/). For a complete example configuration, refer to the [`tempo.yaml` example](https://github.com/grafana/tempo/blob/main/example/docker-compose/single-binary/tempo.yaml) in the Tempo repository.

1. Point the 3.0 deployment at the same object storage bucket or container as your 2.x deployment. This lets the new deployment query historical blocks immediately.

1. Scale compactors to zero in the 3.0 deployment. Only one set of compactors can safely write to shared storage at a time. Running two sets risks data corruption.

   For Helm, set `compactor.replicas` to `0` in your values file:

   ```yaml
   compactor:
     replicas: 0
   ```

   For Docker Compose, stop the compactor service:

   ```bash
   docker compose stop compactor
   ```

1. Validate that the 3.0 deployment can query historical data. Run a trace lookup against the 3.0 query frontend:

   ```bash
   curl http://<TEMPO_3_QUERY_FRONTEND>:3200/api/traces/<TRACE_ID>
   ```

   If the query returns trace data, the new deployment is reading from storage correctly.

1. Verify that Tempo 3.0 connects to Kafka successfully. Confirm that `tempo_distributor_kafka_write_bytes_total` is increasing, `tempo_block_builder_fetch_records_total` is increasing, and `tempo_live_store_ready` equals `1`. The log message `"live-store ready to serve queries"` confirms the live-store is ready.

Before proceeding, confirm all components are healthy and the metrics above show expected values.

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

1. Scale compactors back up in the 3.0 deployment as soon as the 2.x deployment stops writing blocks. The blocklist poll interval (default 5 minutes) controls how quickly queriers discover new blocks. Running without compaction for extended periods can cause the blocklist to grow.

   For Helm, set `compactor.replicas` back to your desired count. For Docker Compose, restart the compactor service:

   ```bash
   docker compose start compactor
   ```

1. Keep the 2.x deployment idle for at least one week. This provides a fallback period to verify the 3.0 deployment is stable.

1. After the validation period, scale all 2.x components to zero and decommission the old infrastructure. Don't delete the shared object storage.

1. Review your compactor replica count. The overlap period may have produced additional blocks that require compaction. Once the backlog clears, reduce compactors to your steady-state configuration.

## Verify the migration

After completing the migration, confirm the following:

- Distributors are writing to Kafka. Verify that `tempo_distributor_kafka_write_bytes_total` is increasing and `tempo_distributor_kafka_write_latency_seconds` is within an acceptable range.
- Block-builders are consuming from Kafka and building blocks. Verify that `tempo_block_builder_fetch_records_total` is increasing and `tempo_block_builder_fetch_errors_total` is at zero.
- Live-stores are serving recent-data queries. Verify that `tempo_live_store_ready` equals `1` and `tempo_live_store_records_processed_total` is increasing. Confirm `tempo_live_store_records_dropped_total` is at zero.
- Kafka lag is within an acceptable range. Check `tempo_ingest_group_partition_lag_seconds`.
- Historical queries return results from object storage.
- Compactors are running in the 3.0 deployment and compacting blocks.
- If you use the metrics-generator, it's producing metrics from Kafka-sourced trace data.

## Roll back

Rolling back depends on how far the migration has progressed.

### Before switching traffic

The 2.x deployment is still active and serving all traffic. No rollback is needed. Stop the 3.0 deployment.

### After switching traffic, before decommissioning 2.x

Revert the traffic routing change to point clients back to the 2.x deployment. Because the cutover is a routing change (DNS, load balancer, or collector config), this is near-instant. Both deployments share the same object storage, so historical data remains available. Traces ingested by the 3.0 deployment into Kafka may not be queryable from the 2.x deployment if block-builders haven't flushed them to storage yet.

### After decommissioning 2.x

Rolling back requires redeploying the 2.x infrastructure. Blocks written by the 3.0 block-builders remain in object storage and are readable by both versions, assuming they use a compatible block format (vParquet4 or later).

{{< admonition type="warning" >}}
There's no downgrade path from Tempo 3.0 to 2.x in terms of the ingest pipeline. After you've committed to the Kafka-based architecture, plan to stay on 3.0. The rollback options above cover the transition period only.
{{< /admonition >}}

## Troubleshoot the migration

This section covers common issues you might encounter during the migration.

### Kafka connection failure

Tempo logs errors such as `"the Kafka address has not been configured"`, `"ping kafka; will retry"`, or `"kafka broker not ready after 10 retries"`.

To resolve this issue:

- Verify that `ingest.kafka.address` in your configuration points to the correct broker address.
- Confirm the broker is reachable from the Tempo deployment. Check network connectivity and firewall rules.
- If you use SASL authentication, verify that both `sasl_username` and `sasl_password` are set. Setting only one produces the error `"the SASL username and password must be both configured"`.

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

## Next steps

- Refer to [Upgrade your Tempo installation](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/upgrade/) for version-specific breaking changes and removed configuration parameters.
- Refer to [Tempo architecture](/docs/tempo/<TEMPO_VERSION>/introduction/architecture/) for details on how block-builders, live-stores, and Kafka interact.
- Refer to [Configuration](/docs/tempo/<TEMPO_VERSION>/configuration/) for the full configuration reference, including all `ingest:`, `block_builder:`, and `live_store:` parameters.
- Refer to [Plan your deployment](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/plan/) for guidance on deployment modes and sizing in Tempo 3.0.
