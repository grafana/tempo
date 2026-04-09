---
title: Partition ring
menuTitle: Partition ring
description: How Tempo manages partition ownership and lifecycle.
weight: 300
topicType: concept
versionDate: 2026-03-20
---

# Partition ring

The partition ring is the mechanism Tempo uses to track which partitions exist, their current state, and which components own them.
By default, the partition ring propagates across the cluster via memberlist gossip and is central to how distributors, live-stores, and block-builders coordinate.

## Tempo partitions vs Kafka partitions

Tempo maintains its own concept of partitions that are logically distinct from Kafka partitions.
While there's typically a 1:1 mapping,
the partition ring gives Tempo independent control over partition states (pending, active, inactive),
ownership (which live-store owns each partition),
and lifecycle management (creating, activating, and deactivating partitions without modifying Kafka's configuration).

## Partition states

Each partition in the ring has one of three states.

### Pending

Pending is the initial state when a new partition is created. No reads or writes occur.

A partition enters pending state when a new live-store starts and creates a partition that doesn't yet exist in the ring.
It stays in pending until enough owners have registered and a minimum waiting period has elapsed,
at which point the owning live-store automatically promotes it to active.

### Active

Active is the normal operating state. Distributors write data to active partitions, and queriers read from them.

A partition transitions from pending to active once enough owners have registered for that partition and a configurable waiting duration has elapsed.
This ensures that all availability zones have had time to register their live-store instances before traffic starts flowing.

### Inactive

Inactive is the read-only state. Distributors stop writing to inactive partitions, but queriers can still read from them.

A partition is marked inactive when scaling down.
It must remain in this state long enough for the block-builder to flush all remaining data for this partition to object storage,
and for queriers to stop relying on the live-store for this partition's recent data.

After this grace period, you can safely remove the partition and its owning live-store.

## Ownership model

### Live-stores

Each Tempo partition is owned by one live-store per availability zone.
In a zone-aware deployment with two zones, each partition has two owners—one per zone.
Both consume the same Kafka partition independently.

When a live-store starts, it checks the ring for its assigned partition.
If the partition exists, it joins as an owner.
If not, it creates the partition in pending state and waits for enough owners to register.

### Distributors

Distributors read the partition ring to determine which partitions are active.
They only send data to active partitions. The ring tells distributors which Kafka partitions to write to.

### Block-builders

Each block-builder instance computes which Kafka partitions it owns based on its ordinal ID and the `partitions_per_instance` setting.
The partition ring indirectly affects block-builders because it determines which partitions receive data from distributors.

## Scaling

### Scaling up

To scale up, deploy a new live-store instance.
The live-store creates a new partition in the ring (pending state).
After enough owners register and the waiting period elapses, the partition transitions to active,
and distributors begin writing to the new partition.

A corresponding Kafka partition must exist. Add Kafka partitions first if needed.

### Scaling down

To scale down, mark the target partition as inactive while the live-store is still running.
Distributors stop writing to it.
Wait for the block-builder to flush remaining data to object storage, then remove the live-store instance.
The partition is eventually cleaned up from the ring.

Skipping the inactive step and abruptly removing a live-store causes recent data for that partition to become temporarily unavailable (unless a zone-aware replica exists).

## Memberlist propagation

The partition ring state is propagated using memberlist, which uses a gossip protocol.
Changes to the ring (new partitions, state transitions) propagate across the cluster within seconds under normal conditions.

During network partitions or high cluster churn, propagation may be delayed.
This can cause brief inconsistencies where different components have different views of the ring.
Tempo handles this gracefully: distributors write to a partition that a live-store hasn't yet seen results in data that's picked up after the live-store catches up,
and queriers contacting a live-store for a partition it doesn't own yet get an empty response,
with the data eventually available from another live-store or from object storage.

## Related resources

Refer to the [memberlist configuration](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#memberlist) for ring propagation settings
and the [ingest configuration](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#ingest) for partition-related settings.
