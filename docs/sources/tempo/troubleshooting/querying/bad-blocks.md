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

A block can get corrupted if the ingester crashed while flushing the block to the backend.

## Fixing bad blocks

The `gen index` and `gen bloom` CLI commands that were previously used to fix corrupted blocks have been removed in Tempo 3.0. These commands were specific to the v2 block format, which is no longer supported.

If you encounter corrupted blocks, the recommended approach is to delete the affected blocks, which may result in some loss of data. Alternatively, you can restore the blocks from a backup if available.

## Removing bad blocks

If the above step on fixing bad blocks reveals that the data file is corrupt, the only remaining solution is to delete
the block, which can result in some loss of data.

The mechanism to remove a block from the backend is backend-specific, but the block to remove will be at:

```
<tenant ID>/<block ID>
```
