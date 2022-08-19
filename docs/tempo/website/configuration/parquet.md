---
title: Parquet
weight: 75
---

# Apache Parquet backend

Tempo now has a columnar block format based on Apache Parquet.
A columnar block format may result in improved search performance and also enables a large ecosystem of tools access to the underlying trace data.

<span style="background-color:#f3f973;">This is an experimental feature. For more information about how to enable it, continue reading.</span>

For more information, refer to the [Parquet design document](https://github.com/mdisibio/tempo/blob/design-proposal-parquet/docs/design-proposals/2022-04%20Parquet.md) and [Issue 1480](https://github.com/grafana/tempo/issues/1480).

## Considerations

While Parquet can be used as a drop-in replacement, Parquet requires more CPU and memory resources.
No data conversion or upgrade process is necessary.
Once Parquet is enabled, Tempo starts writing new data in the new format.
Existing data are left as-is. 

## Enable Parquet 

To use Parquet, set the block format option to `vParquet` in the Storage section of the configuration file.

```yaml
# block format version. options: v2, vParquet
[version: vParquet | default = v2]
```

## Parquet configuration parameters

Some parameters in the Tempo configuration are specific to Parquet.  
For more information, refer to the [configuration documentation](https://grafana.com/docs/tempo/latest/configuration/#storage).

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
