---
title: Bad blocks
description: Troubleshoot queries failing with an error message indicating bad blocks.
weight: 475
aliases:
  - ../../operations/troubleshooting/bad-blocks/ # https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/troubleshooting/bad-blocks/
  - ../bad-blocks/ # https://grafana.com/docs/tempo/<TEMPO_VERSION>/troubleshooting/bad-blocks/
---

# Bad blocks

Queries fail with an error message containing:

```
error querying store in Querier.FindTraceByID: error using pageFinder (1, 5927cbfb-aabe-48b2-9df5-f4c3302d915f): ...
```

This might indicate that there is a bad (corrupted) block in the backend.

## How blocks can get corrupted

Blocks are created by the block-builder, which consumes data from Kafka and flushes blocks to object storage. The block-builder is designed to be recoverable at every stage.
The block-builder rewinds to the last Kafka commit on each cycle, clears its scratch disk, and uses deterministic block IDs so that partial flushes can be safely overwritten.

A block becomes live only once its `meta.json` is written to object storage. Before that point, any crash is fully recoverable. 
In rare cases, corruption can still occur. For example, if object storage acknowledges a write that is not fully persisted, or if the data files are corrupted during upload.

## Removing bad blocks

If you encounter corrupted blocks, delete the affected blocks, which may result in some loss of data. 
The block-builder will replay from Kafka and rebuild any data that hasn't been committed yet. Alternatively, you can restore the blocks from a backup, if available.

The mechanism to remove a block from the backend is backend-specific, but the block to remove will be at:

```
<tenant ID>/<block ID>
```
