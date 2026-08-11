---
title: Kafka
menuTitle: Kafka
description: How Tempo uses Kafka as a durable write-ahead log.
weight: 200
topicType: concept
versionDate: 2026-03-20
---

# Kafka

In [microservices mode](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/deployment-modes/), Tempo uses a Kafka-compatible message queue as the backbone of its write path. Any Kafka-compatible system works.

Kafka isn't used in [monolithic mode](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/deployment-modes/). In monolithic mode, the distributor pushes data in-process directly to the live-store and metrics-generator.

## Role in the architecture

Kafka serves as a durable write-ahead log (WAL) between distributors and downstream consumers (block-builders, live-stores, and metrics-generators).

With Kafka, write-path durability is centralized.
After Kafka acknowledges a write,
Tempo components can recover from a restart as long as the records required for recovery remain available within the Kafka retention window.
Block-builders resume from their last committed Kafka offsets.
Live-stores reload their local WAL and completed local blocks,
then resume Kafka consumption from a point based on their committed offsets and configured replay window.
If local live-store data is missing,
they replay Kafka over that window to rebuild recent-query state.
Because Kafka provides durability,
Tempo doesn't need to replicate data across multiple instances on the write path,
enabling a replication factor of 1 that significantly reduces storage costs.

## Partitioning

Kafka topics are divided into partitions.
Distributors hash the trace ID to determine the target active partition.
Each active Kafka partition is consumed by exactly one block-builder instance and one live-store instance per availability zone.

Tempo maintains its own partition ring that maps Tempo partitions to Kafka partitions. While these are typically 1:1, the partition ring is logically independent from Kafka's partition metadata. Refer to the [partition ring](../partition-ring/) documentation for details.

### Scaling partitions

The number of Kafka partitions determines the maximum parallelism for block-builders and live-stores.
Each active partition is owned by exactly one instance of each consumer type.
The topic can have significantly more partitions than Tempo currently has active.

To scale within the topic's existing capacity,
increase the live-store replicas and the corresponding block-builder capacity without changing the Kafka topic.
If the target active partition count exceeds the topic partition count,
add Kafka partitions before scaling Tempo.
Block-builders and live-stores use partition assignment based on their instance ordinals,
but extra Kafka partitions remain idle until live-stores activate the corresponding Tempo partitions.

## Consumer groups

Tempo runs multiple independent consumer groups against the same Kafka topic:

| Consumer group | Component | Purpose |
|---|---|---|
| `block-builder` | Block-builder | Builds blocks for long-term storage |
| `live-store` | Live-store | Serves recent data for queries |
| `metrics-generator` | Metrics-generator | Derives metrics from trace data |

Each consumer group tracks its own offsets. Block-builders and live-stores consume the same data independently and at their own pace. A slow block-builder doesn't affect live-store availability, and vice versa.

## Retention and offset management

Kafka's retention policy determines how far back consumers can replay. Set it high enough to cover the block-builder's consumption cycle time (plus buffer for failures and restarts) and the live-store's replay window on startup.

If a consumer falls behind Kafka's retention window, it loses the ability to replay missed data. Monitor consumer lag to avoid this situation.

### Key metrics for monitoring consumer lag

```
tempo_ingest_group_partition_lag{group="<consumer-group>"}
tempo_ingest_group_partition_lag_seconds{group="<consumer-group>"}
```

`tempo_ingest_group_partition_lag` tracks lag in number of records per partition. `tempo_ingest_group_partition_lag_seconds` tracks lag in wall-clock seconds.

## Configuration

Kafka connection settings are configured under the `ingest` section:

```yaml
ingest:
  kafka:
    address: kafka:9092
    topic: tempo-traces
```

## Related resources

- [Configure a Kafka-compatible backend](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/configure-kafka/)
- [Ingest configuration reference](/docs/tempo/<TEMPO_VERSION>/configuration/#ingest)
