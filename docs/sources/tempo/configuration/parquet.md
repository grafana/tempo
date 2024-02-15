---
title: Apache Parquet block format
menuTitle: Apache Parquet
description: Learn about Tempo's Parquet block format.
weight: 75
---

# Apache Parquet block format


Tempo has a default columnar block format based on Apache Parquet. This format is required for tags-based search as well as [TraceQL]({{< relref "../traceql" >}}), the query language for traces. The columnar block format improves search performance and enables a large ecosystem of tools to access the underlying trace data.

For more information, refer to the [Parquet design document](https://github.com/grafana/tempo/blob/main/docs/design-proposals/2022-04%20Parquet.md) and [Issue 1480](https://github.com/grafana/tempo/issues/1480).
Additionally, there is now a [Parquet v3 design document](https://github.com/grafana/tempo/blob/main/docs/design-proposals/2023-05%20vParquet3.md).

If you install using the new Helm charts, then Parquet is enabled by default.

## Considerations

The Parquet block format is enabled by default since Tempo 2.0. No data conversion or upgrade process is necessary. As soon as the format is enabled, Tempo starts writing data in that format, leaving existing data as-is.

Block formats based on Parquet require more CPU and memory resources than the previous `v2` format but provide search and TraceQL functionality.

## Choose a different block format

The default block format is `vParquet3`, which is the latest iteration of Tempo's Parquet-based columnar block format.
It introduces dedicated attribute columns, which improve query performance by storing attributes in own columns,
rather than in the generic attribute key-value list.
For more information, see [Dedicated attribute columns]({{< relref "../operations/tempo_cli" >}}).

You can still use the previous format `vParquet2`.
To enable it, set the block version option to `vParquet2` in the Storage section of the configuration file.

```yaml
# block format version. options: v2, vParquet2, vParquet3
[version: vParquet2]
```

In some cases, you may choose to disable Parquet and use the old `v2` block format. Using the `v2` block format disables all forms of search, but also reduces resource consumption, and may be desired for a high-throughput cluster that does not need these capabilities. To make this change, set the block version option to `v2` in the Storage section of the configuration file.

```yaml
# block format version. options: v2, vParquet2, vParquet3
[version: v2]
```

To re-enable the default `vParquet3` format, remove the block version option from the Storage section of the configuration file or set the option to `vParquet3`.

## Parquet configuration parameters

Some parameters in the Tempo configuration are specific to Parquet.
For more information, refer to the [storage configuration documentation]({{< relref "../configuration#storage" >}}).

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
