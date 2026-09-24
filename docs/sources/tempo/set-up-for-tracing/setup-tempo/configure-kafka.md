---
title: Configure a Kafka-compatible backend
menuTitle: Configure Kafka
weight: 500
description: Plan, provision, configure, and validate a Kafka-compatible backend for Tempo microservices mode.
topicType: task
versionDate: 2026-08-10
---

# Configure a Kafka-compatible backend

The microservices mode in Grafana Tempo uses an external Kafka-compatible system as its durable write-ahead log.
The `tempo-distributed` Helm chart configures the connection but doesn't install or operate Kafka.

{{< admonition type="note" >}}
Monolithic mode (`target: all`) doesn't use Kafka.
{{< /admonition >}}

## How Tempo uses Kafka

Tempo uses one topic for trace data from all tenants.
Tempo configures distributors to use the Kafka protocol's strongest acknowledgment mode (`acks=all`)
and waits for the backend to confirm each write before responding to the client.
The topic has three independent consumers:

- Block-builders write trace blocks to object storage and then commit their Kafka offsets.
- Live-stores make recently ingested traces queryable.
- Metrics-generators optionally generate metrics from traces.

## Plan the topic

Create one dedicated topic for Tempo.

For Apache Kafka, use replication factor 3 with a minimum of 2 in-sync replicas is a common starting point.
Adjust these values to your durability target.

### Choose the partition count

The Kafka topic partition count is the upper limit on active Tempo partitions,
not the required live-store or block-builder replica count.
The partition count can be much higher than the number of partitions that Tempo currently has active.
Each active live-store ordinal owns the corresponding Tempo and Kafka partition.
Extra Kafka partitions receive no distributor writes until live-stores scale up and activate the corresponding Tempo partitions.

Estimate the minimum active partitions for peak throughput with:

```text
minimum_active_partitions = ceil(peak_ingestion_bytes_per_second / 10 MB/s)
```

Validate the estimate with your workload.
The topic must have at least as many partitions as the maximum live-store replica count,
but you can provision significantly more for future growth.
Apache Kafka supports increasing an existing topic's partition count,
but not decreasing it in place.
Other Kafka-compatible backends may behave differently.

{{< admonition type="note" >}}
Tempo exposes partition lifecycle APIs that autoscaling controllers can use for safe scale-down.
KEDA can determine the desired replica count,
but safe scale-down also requires a controller to orchestrate the partition lifecycle.
The community `tempo-distributed` chart doesn't yet implement this integration for live-stores and block-builders.
Until that integration is available,
scale these components manually by using the [safe scale-down procedure](#scale-down-a-helm-deployment).
{{< /admonition >}}

When scaling manually,
set the live-store replica count to the desired number of active Tempo partitions,
no higher than the topic partition count.

With the default `block_builder.partitions_per_instance: 1`,
block-builder replicas should track active live-store replicas.
For other values:

```text
block_builder_replicas = ceil(active_partitions / partitions_per_instance)
```

With ordinal-based assignment,
each block-builder claims `partitions_per_instance` contiguous Kafka partitions.
Ensure that all assigned partitions exist:

```text
topic_partitions >= block_builder_replicas * partitions_per_instance
```

A topic pre-provisioned with significantly more partitions than the current active replica count usually already satisfies this requirement.

### Set retention and storage capacity

Set time-based deletion without log compaction, which can remove records before Tempo processes them.

A 24-hour retention period is a sensible starting point for production environments.
The default `live_store.complete_block_timeout` is 20 minutes, and a restarting live-store may replay twice that window.
With this default, don't set retention below 40 minutes.

A shorter retention period leaves block-builders less time to recover and catch up before records become eligible for deletion under the topic retention policy.
Configure retention to exceed the maximum expected block-builder outage and catch-up time,
and ensure that it also covers the live-store replay window.

Estimate raw retained data before replication and compression with:

```text
retained_bytes = sustained_ingestion_bytes_per_second * retention_seconds
```

Account for replication, record overhead, and headroom, and don't let size-based retention shorten the recovery window.

### Set the maximum message size

Tempo can produce Kafka record batches up to `16000000` bytes.
For Apache Kafka, set the topic's `max.message.bytes` to at least `16000000`.
`message.max.bytes` controls the broker-wide default.
Other systems need the equivalent record-batch limit.

## Create the topic

The examples use `tempo-traces`,
which is also the default topic name in the community `tempo-distributed` Helm chart.
If you create a topic with a different name,
set `ingest.kafka.topic` to that exact name in the Tempo configuration.
The configured name must match the topic that you create.

For Apache Kafka, create the topic with a command like this:

```bash
kafka-topics.sh --bootstrap-server <KAFKA_BOOTSTRAP_ADDRESS> \
  --create \
  --topic tempo-traces \
  --partitions <KAFKA_PARTITION_COUNT> \
  --replication-factor 3 \
  --config min.insync.replicas=2 \
  --config cleanup.policy=delete \
  --config retention.ms=86400000 \
  --config max.message.bytes=16000000
```

The Tempo identity needs permissions to:

- Describe the cluster and topic metadata.
- Write to the topic for distributors.
- Read the topic and read and commit consumer-group offsets for block-builders, live-stores, and metrics-generators.

Because these components share the Kafka configuration, one identity commonly has all three permissions.
Auto-creation also requires topic-creation and potentially cluster alter-configuration permissions.
Pre-create the topic and set `auto_create_topic_enabled: false` in production.

## Configure Tempo

Set the bootstrap address and topic in the shared Tempo configuration:

```yaml
ingest:
  kafka:
    address: <KAFKA_BOOTSTRAP_ADDRESS>
    topic: tempo-traces
    auto_create_topic_enabled: false
```

`address` accepts one bootstrap address, after which Tempo discovers the topic's brokers from Kafka metadata.

### Configure authentication and TLS

Tempo supports `PLAIN`, `SCRAM-SHA-256`, and `SCRAM-SHA-512` authentication.
The following example uses SCRAM and TLS:

```yaml
ingest:
  kafka:
    address: <KAFKA_BOOTSTRAP_ADDRESS>
    topic: tempo-traces
    auto_create_topic_enabled: false
    sasl_mechanism: SCRAM-SHA-512
    sasl_username: ${KAFKA_USERNAME}
    sasl_password: ${KAFKA_PASSWORD}
    tls_enabled: true
```

Pass `-config.expand-env=true` when using environment variables.
For private certificate authorities, mount the CA bundle and set `tls_ca_path`; mutual TLS also requires `tls_cert_path` and `tls_key_path`.
Don't use `tls_insecure_skip_verify` in production.

Tempo also supports `OAUTHBEARER` and `AWS_MSK_IAM`.
Refer to the [ingest configuration reference](/docs/tempo/<TEMPO_VERSION>/configuration/#ingest) for their fields.

If you deploy Tempo with the community-maintained `tempo-distributed` Helm chart,
refer to [Get started with Grafana Tempo using Helm](/docs/helm-charts/tempo-distributed/next/get-started-helm-charts)
for Helm values and deployment instructions.

## Validate and monitor

After deploying Tempo:

1. Check for connection, authentication, TLS, and topic metadata errors. Verify each partition's replicas and in-sync replicas.
1. Send a trace through the distributor and query it through the query frontend.
1. Verify that every active partition has a live-store and block-builder owner.

[Tempo Vulture](/docs/tempo/<TEMPO_VERSION>/operations/tempo-vulture/) can validate ingestion and querying continuously.

Collect at least these Tempo metrics:

```promql
# Kafka write throughput.
sum(rate(tempo_distributor_kafka_write_bytes_total[5m]))

# Failed records produced by distributors.
sum by (reason) (rate(tempo_distributor_produce_failures_total[5m]))

# Consumer lag in seconds.
max by (group, partition) (tempo_ingest_group_partition_lag_seconds)

# Block-builder fetch errors.
sum(rate(tempo_block_builder_fetch_errors_total[5m]))

# Live-store readiness.
min(tempo_live_store_ready)
```

Alert before consumer lag approaches topic retention.
Monitor the Kafka backend separately for unavailable or under-replicated partitions, storage, latency, and throttling.

The [Tempo mixin](https://github.com/grafana/tempo/tree/main/operations/tempo-mixin) includes dashboards and alerts for Tempo's Kafka producers and consumers.
It doesn't monitor Kafka broker health; use the monitoring integration for your backend.

## Scale safely

Don't attach a generic HPA directly to live-store or block-builder StatefulSets,
because direct replica reduction bypasses partition draining.

### Scale up

To add active Tempo partitions:

1. Ensure the Kafka topic has at least the target partition count, adding partitions only if needed.
1. Scale live-stores and adjust block-builder capacity for the new active partitions.
1. Verify partition ownership and lag.

Don't scale live-stores beyond the Kafka topic partition count.

### Scale down a Helm deployment

The community `tempo-distributed` chart doesn't currently automate this partition-aware workflow for live-stores and block-builders.
Don't reduce their StatefulSet replicas before draining the partitions selected for removal.

For each highest live-store ordinal that you want to remove:

1. Send a `POST` request directly to that pod's [`/live-store/prepare-partition-downscale`](/docs/tempo/<TEMPO_VERSION>/api_docs/#prepare-live-store-partition-downscale) endpoint.
1. Wait for the block-builder to commit through the partition's remaining Kafka records,
   for the records to reach object storage,
   and for recent queries to stop relying on that live-store.
1. Send a `POST` request to the pod's [`/live-store/prepare-downscale`](/docs/tempo/<TEMPO_VERSION>/api_docs/#prepare-live-store-downscale) endpoint.
1. Reduce `liveStore.replicas`.
1. After the live-store is removed,
   reduce `blockBuilder.replicas` according to `partitions_per_instance`.

If `partitions_per_instance` is greater than 1,
drain every partition assigned to a block-builder before removing that block-builder.
Leave the Kafka topic partition count unchanged during a Tempo scale-down.
Unused partitions remain available for a later scale-up.

Keep live-store StatefulSet names and ordinals stable.
Renaming them changes consumer identities and can trigger replay while local state rebuilds.

## Troubleshoot common failures

| Symptom | Action |
| --- | --- |
| `MESSAGE_TOO_LARGE` or rejected produce requests | Set the backend equivalent of `max.message.bytes` to at least `16000000`. |
| Topic isn't created | Create it explicitly, or enable auto-creation and grant the required permissions. |
| Authentication or TLS errors | Check the SASL mechanism, credentials, CA bundle, client certificates, and broker hostname. |
| A live-store or block-builder doesn't consume | Check partition ownership, topic metadata, Kafka permissions, and fetch errors. |
| Consumer lag grows continuously | Check Tempo resources, object-storage throughput, broker throttling, and partition parallelism. |
| Ingestion fails when Kafka is unavailable | Restore Kafka and ensure trace clients retry failed exports. The default Kafka write timeout is 10 seconds. |

## Related resources

- [Kafka architecture](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/components/kafka/)
- [Partition ring](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/partition-ring/)
- [Ingest configuration reference](/docs/tempo/<TEMPO_VERSION>/configuration/#ingest)
- [Size the Tempo cluster](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/plan/size/)
- [Migrate from Tempo 2.x to 3.0](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/migrate-to-3/)
