---
title: Parquet
weight: 75
---

# Apache Parquet backend

<span style="background-color:#f3f973;">This is an experimental feature released with [Tempo 1.5]({{< relref "../release-notes/v1-5/" >}}). For more information about how to enable it, continue reading.</span>

Tempo now has a columnar block format based on Apache Parquet.
A columnar block format may result in improved search performance and also enables a large ecosystem of tools access to the underlying trace data.

For more information, refer to the [Parquet design document](https://github.com/grafana/tempo/blob/main/docs/design-proposals/2022-04%20Parquet.md) and [Issue 1480](https://github.com/grafana/tempo/issues/1480). Additionally, there is now a [Parquet v3 design document](https://github.com/grafana/tempo/blob/main/docs/design-proposals/2023-05%20vParquet3.md), that is included in Tempo 2.2 as an experimental option.

## Considerations

The new Parquet block format can be used as a drop-in replacement for Tempo's existing block format.
No data conversion or upgrade process is necessary.
As soon as the Parquet format is enabled, Tempo starts writing data in that format, leaving existing data as-is.

Please note, however, that enabling the Parquet block format means Tempo will require more CPU and memory resources than it previously did.


## Enable Parquet

To use Parquet, set the block format option to `vParquet` in the Storage section of the configuration file.

```yaml
# block format version. options: v2, vParquet
[version: vParquet | default = v2]
```

The following adjustments are recommended for your configuration:

```yaml
querier:
  max_concurrent_queries: 100
  search:
    prefer_self: 50   # only if you're using external endpoints

query_frontend:
  max_outstanding_per_tenant: 2000
  search:
    concurrent_jobs: 2000
    target_bytes_per_job: 400_000_000

storage:
  trace:
    <gcs|s3|azure>:
      hedge_requests_at: 1s
      hedge_requests_up_to: 2
```

## Parquet configuration parameters

Some parameters in the Tempo configuration are specific to Parquet.
For more information, refer to the [storage configuration documentation](https://grafana.com/docs/tempo/latest/configuration/#storage).

### Trace search parameters

These configuration options impact trace search.

| Parameter | Default value | Description |
| --- | --- | --- |
| `[read_buffer_size_bytes: <int>]` | `4194304` | Size of read buffers used when performing search on a vParquet block. This value times the `read_buffer_count`  is the total amount of bytes used for buffering when performing search on a Parquet block.
 |
| `[read_buffer_count: <int>]` | 8 | Number of read buffers used when performing search on a vParquet block. This value times the `read_buffer_size_bytes` is the total amount of bytes used for buffering when performing search on a Parquet block.
 |

The `cache_control` section contains the follow parameters for Parquet metadata objects:

| Parameter | Default value | Description |
| --- | --- | --- |
| <code>[footer: <bool> \| default = false]</code> | `false` | Specifies if the footer should be cached |
| `[column_index: <bool> \| default = false]` | `false` | Specifies if the column index should be cached |
| `[offset_index: <bool> \| default = false]` | `false` | Specifies if the offset index should be cached |
