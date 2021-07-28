---
title: Bad Blocks
weight: 475
---

# Bad Blocks

Queries fail with an error message containing:

```
error querying store in Querier.FindTraceByID: error using pageFinder (1, 5927cbfb-aabe-48b2-9df5-f4c3302d915f): ...
```

This might indicate that there is a bad (corrupted) block in the backend.

A block can get corrupted if the ingester crashed while flushing the block to the backend.

## Removing bad blocks

At this moment it's not possible to repair a bad block.
The only solution is to remove the block, which can result in some loss of data.

The mechanism to remove a block from the backend is backend-specific, but the block to remove will be at:

```
<tenant ID>/<block ID>
```
