---
title: Compactor
menuTitle: Compactor
description: How the compactor manages block lifecycle in object storage.
weight: 700
topicType: concept
versionDate: 2026-03-20
---

# Compactor

The compactor maintains the health and efficiency of data in object storage. It compacts small blocks into larger ones, deduplicates overlapping data, and enforces retention policies.

## How it works

### Compaction

Over time, block-builders produce many small blocks. The compactor merges these into fewer, larger blocks. This reduces the number of blocks queriers must consult for a given time range, improves query performance by reducing the search space, and reduces the number of objects in storage (which can lower costs and improve listing performance).

Compaction takes into account which traces exist across blocks to minimize the search space for future queries.

### Deduplication

When block-builders retry a flush (due to a crash or failure), they may produce blocks with overlapping data. The `replaces` field in `meta.json` helps the read path avoid serving duplicate data, but compaction permanently resolves the overlap by merging the blocks.

### Retention

The compactor enforces the configured retention period by deleting blocks older than the retention window from object storage.

## Backend scheduler and worker

The backend scheduler and worker are a forward-looking re-architecture of the compaction process. They improve determinism and remove duplication present in the compactor.

The **scheduler** schedules and tracks jobs assigned to workers. **Workers** connect to the scheduler via gRPC, receive jobs, execute them, and report status. Job types currently include compaction and retention, with more planned for the future.

Workers also maintain the blocklist for all tenants. Tenant polling is coordinated through a ring, the same mechanism previously used by the compactor.

### Transitioning from compactor to scheduler/worker

When transitioning, only one scheduler should be running at a time. Scale workers up as you scale the compactor to 0, to avoid both systems attempting to compact the same blocks.

Where documentation references the compactor for blocklist polling or maintenance, the worker fulfills the same role in scheduler/worker deployments.

## Key metrics

| Metric | Description |
|---|---|
| `tempodb_compaction_blocks_total` | Total blocks compacted |
| `tempodb_compaction_bytes_written_total` | Bytes written during compaction |
| `tempodb_retention_blocks_cleared_total` | Blocks deleted by retention |

## Related resources

Refer to the [compactor configuration](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#compactor) for the full list of options.
