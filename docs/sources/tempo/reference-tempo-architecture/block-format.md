---
title: Block format
menuTitle: Block format
description: How Tempo stores trace data in Apache Parquet blocks.
weight: 500
topicType: concept
versionDate: 2026-03-20
---

# Block format

Tempo stores trace data in Apache Parquet format. Parquet is a columnar storage format that enables efficient querying of specific attributes without reading entire traces.

## Why Parquet

Columnar storage is a natural fit for trace data. A TraceQL query like `{ span.http.status_code = 500 }` only needs to read the `http.status_code` column, not the entire trace. This dramatically reduces the amount of data read from storage. Columnar data also compresses well because values in a column tend to be similar (for example, all service names or all status codes). Parquet is widely supported by data processing tools, making it possible to analyze Tempo data with external tools if needed.

## Block structure

A block is a directory in object storage containing several files.

| File | Purpose |
|---|---|
| `meta.json` | Block metadata: time range, tenant, block ID, `replaces` field for atomic replacement |
| `data.parquet` | Trace data in columnar format |
| Bloom filters | Probabilistic data structures for efficient trace ID lookups |
| Index | Maps trace IDs to row groups within `data.parquet` |
| `nocompact.flg` | Temporary flag preventing compaction (present during block-builder flushes) |

Blocks are stored under `<tenant-id>/<block-id>/` in object storage.

### `meta.json`

The `meta.json` file makes a block "live" — visible to the read path. It contains the block ID, tenant ID, start and end timestamps, total number of objects (traces), and the `replaces` field (a list of block IDs that this block supersedes, used for crash recovery in block-builders).

A block is invisible to queriers until `meta.json` exists. The block-builder uses this property to ensure atomicity during flushes.

## Schema

Tempo uses a span-oriented Parquet schema. Each row represents a span, with columns for intrinsic fields (trace ID, span ID, parent span ID, span name, span kind, span status, duration, start time, root service name, root span name), resource attributes (for example, `service.name`, `deployment.environment`), and span attributes (for example, `http.method`, `http.status_code`).

### Static vs dynamic columns

Attributes are stored in two ways. Static columns are well-known attributes that exist in every trace (for example, `service.name`). These have dedicated columns in the schema and are always efficient to query. Dynamic columns are arbitrary key-value pairs stored in a generic attribute column. Querying these requires scanning the attribute map, which is slower.

### Dedicated attribute columns

You can promote frequently queried attributes to dedicated columns for better performance. This is configured centrally in `storage.trace.block` and applies to all block-producing components (live-store and block-builder).

```yaml
storage:
  trace:
    block:
      parquet_dedicated_columns:
        - name: http.method
          type: string
          scope: span
        - name: deployment.environment
          type: string
          scope: resource
```

Refer to [Dedicated attribute columns](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/dedicated_columns/) for configuration details and recommendations.

## Block format versions

Tempo uses versioned block formats.

| Version | Status |
|---|---|
| vParquet4 | Default in Tempo 3.0 |
| vParquet3 | Supported, readable |
| v2 | Removed in Tempo 3.0 |

The block format is configured in:

```yaml
storage:
  trace:
    block:
      version: vParquet4
```

Existing blocks in older formats remain readable. New blocks are always written in the configured version.

## Related resources

Refer to the [Apache Parquet schema](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/schema/) documentation for the full schema details.
