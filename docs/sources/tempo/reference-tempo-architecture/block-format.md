---
title: Block format
menuTitle: Block format
description: How Tempo stores trace data in Apache Parquet blocks.
weight: 500
topicType: concept
versionDate: 2026-03-20
---

# Block format

Tempo stores trace data in [Apache Parquet](https://parquet.apache.org) format.
Parquet is a columnar storage format that enables efficient querying of specific attributes without reading entire traces.

## Why Parquet

Columnar storage is a natural fit for trace data.
A TraceQL query like `{ span.http.status_code = 500 }` only needs to read the `http.status_code` column, not the entire trace.
This dramatically reduces the amount of data read from storage.
Columnar data also compresses well because values in a column tend to be similar (for example, all service names or all status codes).

## Block structure

A block is a directory in object storage containing several files.

| File            | Purpose                                                                               |
| --------------- | ------------------------------------------------------------------------------------- |
| `meta.json`     | Block metadata: time range, tenant, block ID, `replaces` field for atomic replacement |
| `data.parquet`  | Trace data in columnar format                                                         |
| Bloom filters   | Probabilistic data structures for efficient trace ID lookups                          |
| Index           | Maps trace IDs to row groups within `data.parquet`                                    |
| `nocompact.flg` | Temporary flag preventing compaction (present during block-builder flushes)           |

Blocks are stored under `<tenant-id>/<block-id>/` in object storage.

### `meta.json`

The `meta.json` file makes a block visible to the read path.
It contains the block ID, tenant ID, start and end timestamps, total number of objects (traces).

A block is invisible to queriers until `meta.json` exists.
The block-builder uses this property to ensure atomicity during flushes.

## Schema

Tempo uses a span-oriented heavily nested Parquet schema.
Each row represents a span, with columns for intrinsic fields (trace ID, span ID, parent span ID, span name, span kind, span status, duration, start time, root service name, root span name),
resource attributes (for example, `service.name`, `deployment.environment`), and span attributes (for example, `http.method`, `http.status_code`).

### Intrinsic fields vs generic attributes

A small number of fields are stored as top-level columns in the schema.
These include `service.name` on resources and `status.code` on spans.
Querying these fields is always efficient because they have their own Parquet columns.

All other attributes (both resource and span) are stored in a generic `Attrs` column as key-value pairs.
Querying these requires scanning the attribute map, which is slower.

### Dedicated attribute columns

You can promote frequently queried attributes from the generic column to their own dedicated Parquet columns for better query performance.
This is configured centrally in `storage.trace.block` and applies to all block-producing components (live-store and block-builder).
Dedicated columns are assigned dynamically per block based on the configuration at the time the block is built.

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

| Version   | Status                             |
| --------- | ---------------------------------- |
| vParquet3 | Deprecated in 2.10, removed in 3.0 |
| vParquet4 | Default and latest in Tempo 3.0    |
| vParquet5 | Production-ready, opt-in           |

The block format is configured in:

```yaml
storage:
  trace:
    block:
      version: vParquet4
```

Existing blocks in older formats remain readable. New blocks are always written in the configured version.

## Related resources

Refer to the [Apache Parquet schema](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/schema/) documentation for the full schema details
and the [Apache Parquet block format configuration](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/parquet/) for configuration options.
