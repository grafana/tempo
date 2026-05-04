---
title: Apache Parquet block format
menuTitle: Apache Parquet
description: Learn about Parquet block format in Tempo.
weight: 300
---

# Apache Parquet block format

Tempo has a default columnar block format based on Apache Parquet.
This format is required for tags-based search as well as [TraceQL](../../traceql/), the query language for traces.
The columnar block format improves search performance and enables an ecosystem of tools, including [Tempo CLI](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/tempo_cli/#analyse-blocks), to access the underlying trace data.

## Considerations

The Parquet block format has been the default since Tempo 2.0 and is the only supported block format in Tempo 3.0.

If you install using the [Tempo Helm charts](https://grafana.com/docs/tempo/<TEMPO_VERSION>/setup/helm-chart/), then Parquet is enabled by default.
No data conversion or upgrade process is necessary.
As soon as a block format version is enabled, Tempo starts writing data in that format, leaving existing data as-is.

## Block format versions

{{< admonition type="warning" >}}
The `v2` and `vParquet3` block formats have been removed in Tempo 3.0.
Use `vParquet4` (default) or `vParquet5`.
{{< /admonition >}}

### vParquet4 (default)

`vParquet4` is the default block format in Tempo 3.0.
It introduces columns that support querying array attributes, events, and links.
For more information, refer to [Dedicated attribute columns](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/dedicated_columns/).

### vParquet5

`vParquet5` is production-ready and available as an opt-in alternative to `vParquet4`.
It builds on vParquet4 with the following improvements:

- Expanded dedicated columns: Up to 20 dedicated string columns and 5 dedicated integer columns per scope (span, resource, and event), compared with 10 string columns per scope in vParquet4.
- Event-scoped dedicated columns: Dedicated attribute columns can target event-scoped attributes such as `exception.message`.
- Blob column support: High-cardinality or high-length string attributes (for example, stack traces or UUIDs) can use `zstd` compression instead of dictionary encoding for better efficiency.
- Array-valued dedicated columns: Dedicated columns can store multiple values per attribute using the `options: ["array"]` configuration.

For details on configuring dedicated attribute columns with vParquet5 features, refer to [Dedicated attribute columns](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/dedicated_columns/).

## Change the block format version

To change the block format version, set the `version` option in the [Storage section](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#storage) of the configuration file:

```yaml
storage:
  trace:
    block:
      version: <version>
```

Replace `<version>` with `vParquet4` or `vParquet5`.

To restore the default `vParquet4` format, remove the `version` option from the configuration file or set it to `vParquet4`.

## Parquet configuration parameters

Some parameters in the Tempo configuration are specific to Parquet.
For more information, refer to the [storage configuration documentation](../#storage).

### Trace search parameters

These configuration options impact trace search.

| Parameter | Default value | Description |
|---|---|---|
| `[read_buffer_size_bytes: <int>]` | 10485676 | Size of read buffers used when performing search on a vParquet block. This value times the read_buffer_count is the total amount of bytes used for buffering when performing search on a Parquet block. |
| `[read_buffer_count: <int>]` | 32 | Number of read buffers used when performing search on a vParquet block. This value times the read_buffer_size_bytes is the total amount of bytes used for buffering when performing search on a Parquet block. |

The `cache_control` section contains the follow parameters for Parquet metadata objects:

| Parameter | Default value | Description |
|---|---|---|
| <code>[footer: <bool> \| default = false]</code> | `false`    | Specifies if the footer should be cached    |
| `[column_index: <bool> \| default = false]`   | `false`    | Specifies if the column index should be cached |
| `[offset_index: <bool> \| default = false]`   | `false`    | Specifies if the offset index should be cached |
