---
title: Object storage
menuTitle: Object storage
description: How Tempo uses object storage for long-term trace data retention.
weight: 600
topicType: concept
versionDate: 2026-03-20
---

# Object storage

Object storage is the long-term storage backend for all trace data in Tempo. Block-builders write blocks to it, queriers read from it, and backend workers maintain it.

## Supported backends

Tempo supports three major object storage APIs:

- Amazon S3 (and S3-compatible systems, for example, MinIO)
- Google Cloud Storage (GCS)
- Microsoft Azure Blob Storage

A local filesystem backend is also available for development and testing.

## Storage layout

Data in object storage is organized by tenant and block:

```
<bucket>/
  <tenant-id>/
    <block-id>/
      meta.json
      data.parquet
      bloom-0
      bloom-1
      ...
      index
```

Each tenant has its own directory. Within a tenant, each block is a directory containing the block's files.

### Blocklist

The blocklist is the set of all known blocks for a tenant. Backend workers maintain it by periodically scanning object storage for `meta.json` files and writing a per-tenant block index.

Queriers and query frontends read this tenant index to determine which blocks to search for a given query. They fall back to scanning object storage for `meta.json` files only when the tenant index is unavailable or too stale. The blocklist is distributed across the cluster so that not every component needs to poll storage directly.

### Tenant isolation

Tenants are fully isolated at the storage level. Each tenant's blocks are in a separate directory prefix. There's no cross-tenant data sharing or block merging.

## Durability model

With Tempo 3.0's Kafka-based architecture, durability works in layers.

Kafka provides immediate durability. Once data is acknowledged by Kafka, it's safe even if all Tempo components crash. Object storage provides long-term durability. Once the block-builder flushes a block, the data is durably stored and independent of Kafka. Kafka retention bridges the gap: Kafka retains data long enough for block-builders to consume and flush it. If a block-builder is slow or restarting, Kafka holds the data until it's processed.

There's no single point of failure for data durability. Kafka and object storage together provide end-to-end safety.

## Performance considerations

### Read path

Query performance depends on how efficiently queriers can access blocks. Larger blocks (from compaction) reduce the number of blocks to search but increase individual block read time. Caching bloom filters, Parquet pages, and footers at the querier level significantly reduces object storage reads. Promoting frequently queried attributes to dedicated Parquet columns reduces the amount of data read per query.

### Write path

Block-builder write performance is generally bounded by Kafka consumption rate, local disk speed (blocks are built on scratch disk before upload), and upload bandwidth to object storage.

### Cost

Object storage costs are primarily driven by storage volume (total data retained, controlled by retention period and compaction efficiency) and API operations (GET/PUT/LIST calls). Compaction reduces LIST costs by consolidating blocks. Caching reduces GET costs.

## Configuration

```yaml
storage:
  trace:
    backend: s3  # or gcs, azure, local
    s3:
      bucket: tempo-traces
      endpoint: s3.amazonaws.com
```

## Related resources

Refer to the [storage configuration](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#storage) for the full list of options.
