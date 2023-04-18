---
title: Parquet
menuTitle: Apache Parquet
weight: 75
---

# Apache Parquet block format


Tempo has a default columnar block format based on Apache Parquet. Parquet is required for tags-based search as well as [TraceQL]({{< relref "../traceql" >}}), the query language for traces.

A columnar block format may result in improved search performance and also enables a large ecosystem of tools access to the underlying trace data. For example, you can use `parquet-tools` to [query Parquet data]({{< relref "../operations/parquet" >}}).

For more information, refer to the [Parquet schema]({{< relref "../operations/parquet/schema" >}}) and the [Parquet design document](https://github.com/mdisibio/tempo/blob/design-proposal-parquet/docs/design-proposals/2022-04%20Parquet.md).

If you install using the new Helm charts, then Parquet is enabled by default.

## Considerations

The Parquet block format is enabled by default in Tempo 2.0. No data conversion or upgrade process is necessary. As soon as the format is enabled, Tempo starts writing data in that format, leaving existing data as-is.

Block formats based on Parquet require more CPU and memory resources than the previous `v2` format but provide search and TraceQL functionality.

## Choose a different block format

It is possible to disable Parquet and use the previous `v2` block format. This disables all forms of search, but also reduces resource consumption, and may be desired for a high-throughput cluster that does not need these capabilities. Set the block version option to `v2` in the Storage section of the configuration file.

```yaml
# block format version. options: v2, vParquet, vParquet2
[version: v2]
```

There is also a revised version of the Parquet base block format `vParquet2`. This version improves the interoperability with other tools based on Parquet. `vParquet2` is still experimental and not enabled by default yet. To enable it, set the block format version to `vParquet2` in the Storage section of the configuration file.

```yaml
# block format version. options: v2, vParquet, vParquet2
[version: vParquet2]
```

To re-enable Parquet, set the block version option to `vParquet` in the Storage section of the configuration file.

```yaml
# block format version. options: v2, vParquet, vParquet2
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