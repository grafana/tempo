---
title: Upgrade your Tempo installation
menuTitle: Upgrade
description: Upgrade your Grafana Tempo installation to the latest version.
weight: 510
aliases:
  - ../../setup/upgrade/ # /docs/tempo/<TEMPO_VERSION>/setup/upgrade
---

# Upgrade your Tempo installation

<!-- vale Grafana.We = NO -->
<!-- vale Grafana.Will = NO -->
<!-- vale Grafana.Timeless = NO -->

You can upgrade a Tempo installation to the next version.
However, any release has the potential to have breaking changes.
Before promoting an upgrade to production, test in a non-production environment.

The upgrade process changes for each version, depending upon the changes made for the subsequent release.

This upgrade guide applies to self-managed installations and not for Grafana Cloud.

For information about updating to Tempo 2.x, refer to [Upgrade to Tempo 2.x](/docs/tempo/v2.10.x/set-up-for-tracing/setup-tempo/upgrade/) in the Tempo 2.10 documentation.

For detailed information about any release, refer to the [Release notes](https://grafana.com/docs/tempo/<TEMPO_VERSION>/release-notes/).

{{< admonition type="tip" >}}
You can check your configuration options using the [`status` API endpoint](https://grafana.com/docs/tempo/<TEMPO_VERSION>/api_docs/#status) in your Tempo installation.
{{< /admonition >}}

## Upgrade to Tempo 3.0

Tempo 3.0 is a major release that replaces the ingester-based architecture with a new design that separates the read and write paths.
Block-builders, live-stores, and a backend scheduler replace ingesters and the compactor. For a detailed description of the new architecture, refer to the [Tempo architecture reference](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/).

{{< admonition type="warning" >}}
Tempo 3.0 requires vParquet4 or later as the block format. If your storage configuration specifies vParquet3 or earlier, upgrade the block format before migrating. Refer to [Change the block format version](/docs/tempo/<TEMPO_VERSION>/configuration/parquet/#change-the-block-format-version).
{{< /admonition >}}

The migration path depends on your deployment mode:

- Monolithic mode: Update your configuration (remove `ingester`, `ingester_client`, `compactor`, and `metrics_generator_client` blocks), then upgrade the binary. No Kafka is required. For step-by-step instructions, refer to [Migrate a monolithic deployment](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/migrate-to-3/#migrate-a-monolithic-deployment). For a reference monolithic configuration, refer to [Deploy Tempo locally](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/deploy/locally/).
- Microservices mode: Requires a Kafka-compatible system. Deploy Tempo 3.0 alongside your existing 2.x deployment, switch traffic, then decommission 2.x. Refer to the full [Migrate from Tempo 2.x to 3.0](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/migrate-to-3/) guide.

You can automate configuration migration using the [`tempo-cli migrate config`](/docs/tempo/<TEMPO_VERSION>/operations/tempo_cli/#migrate-config-command) command, which removes obsolete blocks and adds the required `ingest` configuration for microservices mode.

When upgrading to Tempo 3.0, also be aware of these breaking changes:

- No downgrade path: There is no supported downgrade path from 3.0 to 2.x.
- Scalable monolithic mode (SSB) removed: The `scalable-single-binary` target is no longer available. Use either microservices or monolithic (`target: all`) instead. Refer to [Deployment modes](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/deployment-modes/).
- Deployment manifests: Update Helm, Tanka, and other deployment manifests to include the new components and Kafka infrastructure.

### Legacy overrides disabled by default

Tempo now refuses to start if it detects legacy (flat, `unscoped`) overrides in the main configuration or the per-tenant overrides file. [[PR 6741](https://github.com/grafana/tempo/pull/6741)]

To resolve this, either migrate to the scoped `defaults` format (recommended) or temporarily opt back in.

#### Option 1: Migrate to the scoped format

Convert your overrides from the legacy flat format to the scoped `defaults` format. For example:

Before (legacy):

```yaml
overrides:
  ingestion_rate_limit_bytes: 20000000
  ingestion_burst_size_bytes: 20000000
  max_bytes_per_trace: 30000000
  max_traces_per_user: 100000
```

After (scoped):

```yaml
overrides:
  defaults:
    ingestion:
      rate_limit_bytes: 20000000
      burst_size_bytes: 20000000
      max_traces_per_user: 100000
    global:
      max_bytes_per_trace: 30000000
```

You can automate the migration using the Tempo CLI. Refer to the [`tempo-cli migrate overrides-config` command](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/tempo_cli/#migrate-overrides-config-command).

For the full field mapping between legacy and scoped formats, refer to the [Upgrade to Tempo 2.3](#new-defaults-block-in-overrides-module-configuration) section.

#### Option 2: Temporarily opt back in

Set `enable_legacy_overrides: true` in the overrides configuration block or pass `-config.enable-legacy-overrides=true` on the CLI. A deprecation warning is logged on startup and each time per-tenant overrides are loaded. This is a temporary escape hatch. Legacy overrides are removed in a future release.

```yaml
overrides:
  enable_legacy_overrides: true
```

### `mem-ballast-size-mbs` flag removed

The `-mem-ballast-size-mbs` command-line flag has been removed. This flag is no longer needed in Go 1.19 and later, which use `GOMEMLIMIT` instead. [[PR 6403](https://github.com/grafana/tempo/pull/6403)]

If your deployment scripts, Helm values, or Tanka/Jsonnet configurations pass `-mem-ballast-size-mbs`, remove it. Tempo fails to start with an unrecognized flag error.

### Metrics-generator configuration changes

The metrics-generator gRPC endpoint and push path have been removed. In Tempo 3.0, the [metrics-generator](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/components/metrics-generator/) consumes directly from Kafka rather than receiving spans through gRPC from the distributor. [[PR 6618](https://github.com/grafana/tempo/pull/6618)]

If your configuration includes a top-level `metrics_generator_client` block, you can safely remove it. Tempo 3.0 ignores this block, and it is deprecated. It is removed in a future release.

### Block configuration centralized to `storage.trace.block`

Block and WAL configuration for the [block-builder](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/components/block-builder/) and [live-store](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/components/live-store/) is now always sourced from `storage.trace.block`. Per-module block configuration fields have been removed. [[PR 6647](https://github.com/grafana/tempo/pull/6647)]

If your configuration sets block-level options such as `version`, `parquet_dedicated_columns`, or `parquet_row_group_size_bytes` under `block_builder.block` or `live_store.block_config`, move them to `storage.trace.block`.

Before:

```yaml
block_builder:
  block:
    version: "vParquet5"
    parquet_dedicated_columns:
      - { scope: resource, name: service.name, type: string }

live_store:
  block_config:
    version: "vParquet5"
    parquet_dedicated_columns:
      - { scope: resource, name: service.name, type: string }
```

After:

```yaml
storage:
  trace:
    block:
      version: "vParquet5"
      parquet_dedicated_columns:
        - { scope: resource, name: service.name, type: string }
```

### `partition_ring_live_store` removed

Tempo 3.0 removes the top-level `partition_ring_live_store` setting. Tempo now uses a single [partition ring](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/partition-ring/), so this configuration toggle is no longer needed. [[PR 6981](https://github.com/grafana/tempo/pull/6981)]

If your 2.x configuration still includes this field, remove it before or during your 3.0 upgrade.

Before:

```yaml
partition_ring_live_store: true
```

After:

```yaml
# Remove partition_ring_live_store
```

### Live-store and query defaults reduced

The default values for several [live-store](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/components/live-store/) and query-frontend settings have been reduced to produce smaller WAL blocks, release completed blocks sooner, and align the metrics query backend boundary with search.

| Setting                                          | Previous default | New default |
| ------------------------------------------------ | ---------------- | ----------- |
| `live_store.flush_check_period`                  | `10s`            | `5s`        |
| `live_store.max_block_duration`                  | `30m`            | `30s`       |
| `live_store.max_block_bytes`                     | `100 MiB`        | `50 MiB`    |
| `live_store.complete_block_timeout`              | `1h`             | `20m`       |
| `query_frontend.metrics.query_backend_after`     | `30m`            | `15m`       |

If you explicitly set these values in your configuration, no action is needed.

### Ingester removal

The ingester module is removed entirely. All ingester-related configuration fields, CLI flags, alerts, and dashboard panels must be removed from your deployment. The write path is now handled by the [block-builder](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/components/block-builder/) and [live-store](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/components/live-store/).

Removed configuration sections: `ingester`, `ingester_client`, `compactor`, `metrics_generator_client`.
The `ingest.enabled` field is also removed, but the `ingest` block itself is still required for microservices mode (for example, `ingest.kafka`).

For step-by-step migration instructions, refer to [Migrate from Tempo 2.x to 3.0](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/migrate-to-3/).

(PRs [#6959](https://github.com/grafana/tempo/pull/6959), [#6504](https://github.com/grafana/tempo/pull/6504), [#6667](https://github.com/grafana/tempo/pull/6667), [#6873](https://github.com/grafana/tempo/pull/6873))

### Compactor removal and CLI flag changes

The compactor component and the `v2` block encoding are removed. Compaction is now handled by the [backend scheduler and worker](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/components/compaction/), which track job progress centrally and automatically reschedule failed jobs.

Remove all compactor-related configuration, alerts, and dashboard panels from your deployment. The following `tempo-cli` commands are also removed because they were specific to the `v2` format: `list block`, `list index`, `view index`, `gen index`, and `gen bloom`.

The compaction CLI flags drop their duplicate `compaction.` prefix. Update these flags in your configuration:

- `compaction.compaction.block-retention` → `compaction.block-retention`
- `compaction.compaction.max-objects-per-block` → `compaction.max-objects-per-block`
- `compaction.compaction.max-block-bytes` → `compaction.max-block-bytes`
- `compaction.compaction.compaction-window` → `compaction.compaction-window`

(PRs [#6273](https://github.com/grafana/tempo/pull/6273), [#6369](https://github.com/grafana/tempo/pull/6369), [#6909](https://github.com/grafana/tempo/pull/6909))

### `RetryInfo` enabled by default

The `distributor.retry_after_on_resource_exhausted` setting now defaults to `5s` (previously `0`). OTLP clients receive a retry hint on `ResourceExhausted` errors from the [distributor](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/components/distributor/). [[PR 7088](https://github.com/grafana/tempo/pull/7088)]

To disable cluster-wide, set the value to `0`. To disable for a single tenant, set the per-tenant override `ingestion.retry_info_enabled: false`.

### TraceQL array matching changes

The TraceQL AST optimization changes the semantics of `!=` and `!~` operators when used with array attributes. `!=` now means `NOT IN` (previously `CONTAINS NOT EQUAL`) and `!~` now means `MATCH NONE` (previously `CONTAINS NON-MATCH`). Regex operands must be of type string or string array. [[PR 6353](https://github.com/grafana/tempo/pull/6353)]

If you have queries that depend on the previous behavior, disable the optimization with the query hint `skip_optimization=true`.

### Other breaking changes

- The `all` target is now 3.0-compatible and the `scalable-single-binary` target is removed. Refer to [Deployment modes](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/deployment-modes/). [[PR 6283](https://github.com/grafana/tempo/pull/6283)]
- The OpenCensus receiver is removed. Migrate to OTLP. [[PR 6523](https://github.com/grafana/tempo/pull/6523)]
- `SpanMetricsSummary` is removed and querier code simplified. (PRs [#6496](https://github.com/grafana/tempo/pull/6496), [#6510](https://github.com/grafana/tempo/pull/6510))
- The `querier.query_live_store` configuration is removed. [[PR 7048](https://github.com/grafana/tempo/pull/7048)]
- `query_frontend.search.query_ingesters_until` is removed in favor of `query_frontend.search.query_backend_after`. Refer to the [query-frontend](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/components/query-frontend/) component reference. [[PR 6507](https://github.com/grafana/tempo/pull/6507)]
- The `tempo-cli query search` command no longer accepts timestamps without a timezone (for example, `2024-01-01T00:00:00`). Use RFC3339 format (for example, `2024-01-01T00:00:00Z`) or relative time (for example, `now-1h`). Refer to the [Tempo CLI documentation](/docs/tempo/<TEMPO_VERSION>/operations/tempo_cli/). [[PR 6458](https://github.com/grafana/tempo/pull/6458)]
- Tempo 3.0 upgrades to Go 1.26.2. [[PR 6443](https://github.com/grafana/tempo/pull/6443)]


<!-- vale Grafana.We = YES -->
<!-- vale Grafana.Will = YES -->
<!-- vale Grafana.Timeless = YES -->
