---
title: Upgrade your Tempo installation
menuTitle: Upgrade
description: Upgrade your Tempo installation to the latest version.
weight: 75
---

# Upgrade Tempo

You can upgrade an existing Tempo installation to the next version. However, any new release has the potential to have breaking changes that should be tested in a non-production environment prior to rolling these changes to production.

The upgrade process changes for each version, depending upon the changes made for the subsequent release.

This upgrade guide applies to on-premise installations and not for Grafana Cloud.

>**TIP**: You can check your configuration options using the [`status` API endpoint]({{< relref "../api_docs/#status" >}}) in your Tempo installation.

## Upgrade from Tempo 1.5 to 2.0

Tempo 2.0 marks a major milestone in Tempo’s development. When planning your upgrade, consider these factors:

- Breaking changes:
  - Renamed, removed, and moved configurations are described in section below.
  - The `TempoRequestErrors` alert was removed from mixin. Any Jsonnet users relying on this alert should copy this into their own environment.
- Advisory:
  - Changed defaults – Are these updates relevant for your installation?
  - TraceQL editor needs to be enabled in Grafana to use the query editor.
  - Resource requirements have changed for Tempo 2.0 with the default configuration.

Once you upgrade to Tempo 2.0, there is no path to downgrade.

### Check Tempo installation resource allocation

Parquet provides faster search and is required to enable TraceQL. However, the Tempo installation will require additional CPU and memory resources to use Parquet efficiently.

You can can continue using the previous `v2` block format using the instructions provided in the [Parquet configuration documentation]({{< relref "../configuration/parquet/" >}}).

### Enable TraceQL in Grafana

TraceQL is enabled by default in Tempo 2.0. The TraceQL query editor requires Grafana 9.3.2 and later.

The TraceQL query editor is in beta in Grafana 9.3.2 and needs to be enabled with the `traceqlEditor` feature flag.

### Check configuration options for removed and renamed options

The following tables describe the parameters that have been removed or renamed.

#### Removed and replaced

| Parameter | Comments |
| --- | --- |
| <code>query_frontend:<br>&nbsp;&nbsp;query_shards:<code> | Replaced by `trace_by_id.query_shards`. |
| <code>querier:<br>&nbsp;&nbsp;query_timeout:<code> | Replaced by two different settings: `search.query_timeout` and `trace_by_id.query_timeout`. |
| <code>ingester:<br>&nbsp;&nbsp;use_flatbuffer_search:<code> | Removed and automatically determined based on block format. |
| `search_enabled` | Removed. Now defaults to true. |
| `metrics_generator_enabled` | Removed. Now defaults to true. |

#### Renamed

The following `compactor` configuration parameters were renamed.

| Parameter | Comments |
| --- | --- |
| <code>compaction:<br>&nbsp;&nbsp;chunk_size_bytes:<code> | Renamed to `v2_in_buffer_bytes` |
| <code>compaction:<br>&nbsp;&nbsp;flush_size_bytes:<code> | Renamed to `v2_out_buffer_bytes` |
| <code>compaction:<br>&nbsp;&nbsp;iterator_buffer_size:<code> | Renamed to `v2_prefetch_traces_count` |

The following `storage` configuration parameters were renamed.

| Parameter | Comments |
| --- | --- |
| <code>wal:<br>&nbsp;&nbsp;encoding:<code> | Renamed to `v2_encoding` |
| <code>block:<br>&nbsp;&nbsp;index_downsample_bytes:<code> | Renamed to `v2_index_downsample_bytes` |
| <code>block:<br>&nbsp;&nbsp;index_page_size_bytes:<code> | Renamed to `v2_index_page_size_bytes` |
| <code>block:<br>&nbsp;&nbsp;encoding:<code> | Renamed to `v2_encoding` |
| <code>block:<br>&nbsp;&nbsp;row_group_size_bytes:<code> | Renamed to `parquet_row_group_size_bytes` |

The Azure Storage configuration section now uses snake case with underscores (`_`) instead of dashes (`-`). Example of using snake case on Azure Storage config:

```yaml
# config.yaml
storage:
  trace:
    azure:
      storage_account_name:
      storage_account_key:
      container_name:
```