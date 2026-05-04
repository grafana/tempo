---
title: Live-store
menuTitle: Live-store
description: How the live-store serves recent trace data.
weight: 400
topicType: concept
versionDate: 2026-03-20
---

# Live-store

The live-store is the read-path component responsible for serving recent trace data.
It holds traces in memory, making them available for queries during the window between ingestion and block availability in object storage.

How the live-store receives data depends on the [deployment mode](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/deployment-modes/):

- Microservices mode: The live-store consumes trace data from Kafka independently of block-builders.
- Monolithic mode: The live-store receives trace data directly from the distributor in-process. No Kafka consumption is involved.

## Why live-stores exist

In microservices mode, there's a gap between when trace data is written to Kafka and when the block-builder flushes it to object storage. During this window, the only way to query that data is through the live-store.

In monolithic mode, the live-store serves the same role of providing immediate query access to recently ingested data, but it receives data directly from the distributor rather than from Kafka.

In both modes, the live-store holds traces in memory organized by trace ID, responds to queries from queriers for recent data, and periodically flushes traces to a local WAL in Parquet format for TraceQL search and metrics queries.

## Trace lifecycle

When the live-store receives spans, it assembles them into traces in memory.
Each trace goes through three stages.

First, the trace is active—it's receiving spans, remains in memory, and is queryable.
Then, when no new spans have arrived within the configured `max_trace_idle`
the trace becomes idle and is flushed to the local WAL.
Once flushed, the trace data is written in Parquet format and becomes available for TraceQL search.
Eventually, the WAL data is cut into complete blocks.

### Trace idle period

The `max_trace_idle` setting controls how long the live-store waits after the last span arrives before considering a trace idle and flushing it to the WAL.

```yaml
live_store:
  max_trace_idle: 10s
```

Increasing this value keeps traces in memory longer, which improves the chances that all spans for a trace are co-located when flushed.
This is beneficial for [long-running traces](https://grafana.com/docs/tempo/<TEMPO_VERSION>/troubleshooting/querying/long-running-traces/). However, it also increases memory usage.

## Partition ownership

Live-stores own the partition lifecycle within Tempo.
Each live-store instance consumes from one or more Tempo partitions, and each partition is owned by exactly one live-store per availability zone.

### Partition ring

The live-store maintains a partition ring that tracks which Tempo partitions exist,
which live-stores own each partition, and the state of each partition (pending, active, or inactive).

This ring is propagated via memberlist gossip.
Refer to the [partition ring](../partition-ring/) documentation for details on partition states and transitions.

### Startup

When a live-store starts, it checks the partition ring for its assigned partition.
If the partition exists, the live-store joins as an owner.
If it doesn't exist, the live-store creates it in pending state and waits for enough owners to register before automatically promoting it to active.
In microservices mode, the live-store then replays from its last committed Kafka offset to rebuild in-memory state.

### Shutdown and scaling down

Scaling down live-stores requires marking the partition as inactive while the live-store is still running.
This transitions the partition to read-only mode.
After enough time passes for the data to be flushed to object storage,
you can safely remove the partition and live-store.

Abruptly removing a live-store without marking its partition inactive makes that partition's recent data temporarily unavailable until another live-store picks it up (in a zone-aware setup, the other zone's live-store continues serving).

## Zone-aware high availability

For production deployments, live-stores are typically deployed across multiple availability zones.
Each Tempo partition is owned by one live-store per zone.

If a live-store in one zone becomes unavailable, the live-store in the other zone continues serving queries for the same partitions.
Queriers only need a response from one live-store per partition (read quorum of 1),
so queries succeed as long as at least one zone is healthy.
This provides high availability without requiring data deduplication on the read path.

Refer to the [zone-aware live-stores](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/manage-advanced-systems/zone-aware-live-stores/) documentation for configuration details.

## Local WAL

When traces are flushed from memory, they're written to a local WAL in Parquet format. This serves two purposes.

First, it provides search availability—after data is in the WAL,
trace data is available for TraceQL search queries, not just trace ID lookups.
Second, it aids recovery on restart. In microservices mode, if the live-store restarts,
it replays from Kafka, and the WAL provides a way to serve queries during replay.

The WAL is eventually cut into complete blocks that are also stored locally.
These blocks are queryable until the data ages out of the live-store's retention window.

## Key metrics

| Metric | Description |
|---|---|
| `tempo_live_store_traces_created_total` | Total number of traces created in the live-store |
| `tempo_live_store_lagged_requests_total` | Requests where the live-store could not guarantee complete results due to Kafka lag, labeled by `route` |
| `tempo_warnings_total` | Warnings during trace processing, labeled by `reason` |
| `tempo_ingest_group_partition_lag{group="live-store"}` | Consumer lag per partition |

## Related resources

Refer to the [live-store configuration](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#live-store) for the full list of options.
