---
aliases:
- /docs/tempo/v1.2.1/troubleshooting/bad-blocks/
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

## Fixing bad blocks

At the moment, a backend block can be fixed if either the index or bloom-filter is corrupt/deleted.

To fix such a block, first download it onto a machine where you can run the `tempo-cli`.

Next run the `tempo-cli`'s `gen index` / `gen bloom` commands depending on which file is corrupt/deleted.
The command will create a fresh index/bloom-filter from the data file at the required location (in the block folder).
To read all the options for this command, check the [cli docs](../../operations/tempo_cli).

Finally, upload the generated index or bloom-filter onto the object store backend under the folder for the block.

## Removing bad blocks

If the above step on fixing bad blocks reveals that the data file is corrupt, the only remaining solution is to delete
the block, which can result in some loss of data.

The mechanism to remove a block from the backend is backend-specific, but the block to remove will be at:

```
<tenant ID>/<block ID>
```
