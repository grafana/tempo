---
title: Block-builder
menuTitle: Block-builder
description: How the block-builder consumes from Kafka and builds blocks for long-term storage.
weight: 300
topicType: concept
versionDate: 2026-03-20
---

# Block-builder

The block-builder is the write-path component responsible for building Parquet blocks and flushing them to object storage.
It consumes trace data from Kafka and organizes it into blocks suitable for long-term retention and efficient querying.

The block-builder only runs in [microservices mode](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/deployment-modes/). In monolithic mode, the live-store handles flushing trace data to object storage directly.

For a configuration block example, refer to the [block-builder section](/docs/tempo/<TEMPO_VERSION>/configuration/#block-builder) of the Configuration documentation.

## Consumption cycle

The block-builder operates on a cyclical consumption model.

On each cycle, the block-builder rewinds to the last committed Kafka offset to ensure any partially processed data from a previous cycle is re-consumed.
It reads records from Kafka up to a configured boundary (time-based), organizes the consumed spans by tenant,
writes them into Parquet blocks on local disk, uploads the blocks to object storage, and commits the Kafka offset.

### Hard cuts

The block-builder performs a hard cut at the end of each consumption cycle.
All spans consumed during that cycle are flushed into blocks, regardless of whether the traces they belong to are complete.
If a trace has spans arriving across two consumption cycles, those spans end up in separate blocks.

This is by design. The block-builder has no concept of "live traces" or trace completion.
Trace assembly is handled at query time by the querier, which merges spans from multiple blocks.

## Block creation

Each consumption cycle produces one or more blocks per tenant per partition.
Blocks are written in Apache Parquet format and contain the span data (`data.parquet`),
block metadata (`meta.json`) including time range, tenant, and a `replaces` field for atomic block replacement,
as well as bloom filters and indexes for efficient querying.

### Deterministic block IDs

The block-builder generates block IDs deterministically based on the partition, tenant, and Kafka offset range.
This is critical for crash recovery: if a block-builder crashes mid-flush and restarts,
it produces the same block IDs on retry, safely overwriting any partial data from the previous attempt.

## Flush and recovery

The flush process supports safe replay at every stage.

### Flush order

The block-builder flushes blocks to object storage in a specific order:

1. Bloom filters and indexes
1. `data.parquet`
1. `nocompact.flg` (a flag file that prevents compaction during the flush)
1. `meta.json` (the block becomes "live" at this point)

A block isn't visible to the read path until its `meta.json` is written.
Before that point, any crash is fully recoverable—the block-builder rewinds and overwrites.

### Recovering from partial flushes

If the block-builder crashes before writing `meta.json`, the block is invisible to readers.
On restart, it rewinds to the last committed offset, regenerates the same block ID, and overwrites the partial data.

If the crash happens after `meta.json` is written, the block is already live.
On restart, the block-builder detects the existing block and advances to the next ID in sequence,
using the `replaces` field to atomically replace the old block.

### The `replaces` field

When a block-builder retries a flush and finds that a previous block already exists (its `meta.json` was written),
the new block includes a `replaces` field in its `meta.json` listing the old block ID.
This tells the read path to ignore the old block once the new one is visible,
preventing duplicate data from appearing in query results.

### The `nocompact.flg` file

The `nocompact.flg` file is written before `meta.json` to prevent backend workers from touching the block while it's still being built.
After the block-builder finishes its cycle, it removes this flag.
This prevents a race condition where a backend worker might try to compact a block that's about to be replaced.

## Scaling

Each block-builder instance consumes from one or more Kafka partitions.
The maximum number of block-builder instances equals the number of Kafka partitions.

Block-builders use static partition assignment. Kafka does not move partitions between consumers in the consumer group for this component.
There are two ways to assign partitions:

- `partitions_per_instance`: Each instance computes which partitions it owns based on its ordinal ID.
  This is the default and works well with `StatefulSets` where the block-builder mirrors its replica count from the live-store, scaling in lockstep.
- `assigned_partitions`: An explicit mapping of instance IDs to partition lists. This gives full manual control over which instance handles which partitions.

Size the scratch disk to hold at least one full consumption cycle's worth of data across all assigned partitions and tenants.

## Key metrics

| Metric | Description |
|---|---|
| `tempo_block_builder_flushed_blocks` | Number of blocks flushed to object storage |
| `tempo_block_builder_fetch_errors_total` | Kafka fetch errors encountered |
| `tempo_ingest_group_partition_lag{group="block-builder"}` | Consumer lag per partition |

## Related resources

Refer to the [block-builder section of the Tempo configuration](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#block-builder) for the full list of block-builder options.
