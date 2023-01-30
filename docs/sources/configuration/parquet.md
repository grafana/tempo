---
title: Parquet
menuTitle: Apache Parquet
weight: 75
---

# Apache Parquet block format


Tempo has a default columnar block format based on Apache Parquet. [TraceQL]({{< relref "../traceql" >}}), the query language for traces, requires Parquet. In a future release, Parquet will be required to use traces search.

A columnar block format may result in improved search performance and also enables a large ecosystem of tools access to the underlying trace data.

For more information, refer to the [Parquet design document](https://github.com/mdisibio/tempo/blob/design-proposal-parquet/docs/design-proposals/2022-04%20Parquet.md) and [Issue 1480](https://github.com/grafana/tempo/issues/1480).

If you install using the new Helm charts, then Parquet is enabled by default.

## Considerations

The new Parquet block is enabled by default in Tempo 2.0. No data conversion or upgrade process is necessary. As soon as the Parquet format is enabled, Tempo starts writing data in that format, leaving existing data as-is.

The new Parquet block format requires more CPU and memory resources than the previous v2 format but provides faster search and the new TraceQL.

## Disable Parquet

It is possible to disable Parquet and use the previous v2 block format. This disables TraceQL and will result in reduced search performance, but also reduces resource consumption, and may be desired for a high-throughput cluster that does not need these capabilities. Set the block format option to v2 in the Storage section of the configuration file.

```yaml
# block format version. options: v2, vParquet
[version: v2]
```

To re-enable Parquet, set the block format option to `vParquet` in the Storage section of the configuration file.

```yaml
# block format version. options: v2, vParquet
[version: vParquet]
```

## Parquet configuration parameters

Some parameters in the Tempo configuration are specific to Parquet.
For more information, refer to the [storage configuration documentation](https://grafana.com/docs/tempo/latest/configuration/#storage).

### Trace search parameters

These configuration options impact trace search.

| Parameter | Default value | Description |
| --- | --- | --- |
| `[read_buffer_size_bytes: <int>]` | `10485676` | Size of read buffers used when performing search on a vParquet block. This value times the `read_buffer_count`  is the total amount of bytes used for buffering when performing search on a Parquet block.
 |
| `[read_buffer_count: <int>]` | 32 | Number of read buffers used when performing search on a vParquet block. This value times the `read_buffer_size_bytes` is the total amount of bytes used for buffering when performing search on a Parquet block.
 |

The `cache_control` section contains the follow parameters for Parquet metadata objects:

| Parameter | Default value | Description |
| --- | --- | --- |
| <code>[footer: <bool> \| default = false]</code> | `false` | Specifies if the footer should be cached |
| `[column_index: <bool> \| default = false]` | `false` | Specifies if the column index should be cached |
| `[offset_index: <bool> \| default = false]` | `false` | Specifies if the offset index should be cached |

## Convert to Parquet

If you have used an earlier version of the Parquet format, you can use `tempo-cli` to convert a Parquet file from its existing schema to the one used in Tempo 2.0.

For instructions, refer to the [Parquet convert command documentation]({{< relref "../operations/tempo_cli#parquet-convert-command" >}}).