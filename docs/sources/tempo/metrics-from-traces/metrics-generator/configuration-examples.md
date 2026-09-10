---
title: Metrics-generator configuration examples
menuTitle: Configuration examples
description: Complete, valid metrics-generator configuration blocks for common setups, from a minimal enablement to a cost-optimized and a full-featured configuration.
weight: 150
---

# Metrics-generator configuration examples

The [configuration reference](/docs/tempo/<TEMPO_VERSION>/configuration/#metrics-generator) documents each metrics-generator option individually.
This page shows complete, valid configuration blocks for common setups, so you can copy a whole configuration and adapt it instead of assembling one option at a time.
For architecture and what each processor emits, refer to [Metrics-generator](/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/metrics-generator/).

Each example is a complete, working starting point:

- How to [send metrics to a remote Prometheus or Mimir endpoint](#send-metrics-to-a-remote-prometheus-or-mimir-endpoint), including Grafana Cloud.
- A [minimal configuration](#minimal-configuration) that enables the generator with defaults.
- A [cost-optimized configuration](#cost-optimized-configuration) that reduces active series and remote-write volume.
- A [full-featured configuration](#full-featured-configuration) that adds custom dimensions, dimension mappings, and filter policies.

## Before you begin

The metrics-generator requires the following:

- The metrics-generator target must be deployed and running.
- A Prometheus-compatible `remote_write` endpoint to receive metrics, such as Prometheus, Grafana Mimir, or Grafana Cloud Metrics.
- At least one enabled processor. Processors are disabled by default, so the generator produces no metrics until you enable them.

The generator derives metrics only from spans as they're ingested.
It can't backfill metrics from traces that were ingested before you enabled it, so metrics start from the moment you turn a processor on.

{{< admonition type="note" >}}
Enabling metrics generation produces extra active series, which can affect cost.
Start with the minimal configuration, confirm the output, then add dimensions and processors incrementally.
{{< /admonition >}}

## Where configuration lives

Metrics-generator settings are split across two places in `tempo.yaml`.
Putting a setting in the wrong block is a common source of errors, so check the location before you copy an option.

| Block | Purpose | Examples |
| ----- | ------- | -------- |
| `metrics_generator` (top level) | Infrastructure that applies to the whole generator. | `storage.remote_write`, `registry.collection_interval`, ring settings. |
| `overrides` | Which processors are enabled and how each processor behaves. Set globally under `overrides.defaults` or per tenant. | `processors`, `processor.span_metrics`, `processor.service_graphs`, `max_active_series`, `generate_native_histograms`. |

Per-processor tuning, such as `dimensions`, `filter_policies`, `histogram_buckets`, `intrinsic_dimensions`, and `dimension_mappings`, lives in a `processor.<processor>` block, under either `metrics_generator.processor` or `overrides...metrics_generator.processor`.
The examples on this page use the `overrides` block, which is the recommended place because it also supports per-tenant settings.
`processors`, which selects which processors are enabled, is set only in the `overrides` block.

{{< admonition type="warning" >}}
Use the current overrides format, which nests tenant settings under `overrides.defaults` (global) and `overrides.<tenant-id>` (per tenant).
The flat legacy overrides format is deprecated, is disabled by default (`enable_legacy_overrides: false`), and will be removed in a future release.
Mixing the two formats in one file produces configuration that doesn't take effect as expected.
{{< /admonition >}}

## Send metrics to a remote Prometheus or Mimir endpoint

Every example on this page includes a `storage.remote_write` block that writes to a local Prometheus URL.
Replace that block when you send generated metrics to Grafana Cloud or another remote Prometheus-compatible endpoint.

```yaml
metrics_generator:
  storage:
    path: /var/tempo/generator/wal
    remote_write:
      # Grafana Cloud Metrics. For Mimir use /api/v1/push.
      # For Prometheus use /api/v1/write.
      - url: https://<prometheus-host>/api/prom/push
        send_exemplars: true
        basic_auth:
          username: <instance-id>
          password: <api-token>
```

Use this `storage.remote_write` block in place of the local Prometheus URL in any of the examples that follow.
Change the URL path to match the destination: Grafana Cloud Metrics uses `/api/prom/push`, Mimir uses `/api/v1/push`, and Prometheus uses `/api/v1/write`.
If you enable the metrics-generator through Grafana Cloud, refer to the [Metrics-generator in Grafana Cloud](https://grafana.com/docs/grafana-cloud/send-data/traces/metrics-generator/) documentation for Cloud-specific defaults and enablement.

## Minimal configuration

This configuration enables the span-metrics and service-graphs processors with their default settings and remote-writes the results.
It's the smallest complete configuration that produces metrics.

```yaml
# tempo.yaml
metrics_generator:
  storage:
    path: /var/tempo/generator/wal
    remote_write:
      - url: http://prometheus:9090/api/v1/write
        send_exemplars: true

overrides:
  defaults:
    metrics_generator:
      processors:
        - span-metrics
        - service-graphs
```

With this configuration:

- The generator processes spans of every kind. No default filter excludes any span kind.
- The span-metrics processor emits `traces_spanmetrics_calls_total`, `traces_spanmetrics_latency`, and `traces_spanmetrics_size_total`.
- The service-graphs processor emits the `traces_service_graph_*` metrics.

## Cost-optimized configuration

This configuration reduces active series and remote-write volume.
Use it when the default configuration produces more series, or costs more, than you want.

```yaml
# tempo.yaml
metrics_generator:
  storage:
    path: /var/tempo/generator/wal
    remote_write:
      - url: http://prometheus:9090/api/v1/write
  registry:
    # Collect and remote-write less frequently. Default is 15s.
    # The accepted range is 15s to 5m.
    collection_interval: 30s

overrides:
  defaults:
    metrics_generator:
      processors:
        - span-metrics
        - service-graphs
      # Per-instance cap. A value of 0 disables the check.
      max_active_series: 10000
      # Native histograms use far fewer active series than classic histograms.
      # The receiving endpoint must be configured to ingest native histograms.
      generate_native_histograms: native
      processor:
        span_metrics:
          intrinsic_dimensions:
            # span_name is the largest cardinality driver. Disabling it
            # collapses per-operation series into per-service series.
            span_name: false
            span_kind: false
```

This configuration applies several independent reductions.
Apply only the ones that fit your needs:

- **Native histograms**: `generate_native_histograms: native` replaces the classic per-bucket series with a single native-histogram series for both the span-metrics and service-graphs histograms.
  The receiving endpoint must be configured to ingest native histograms.
  Refer to [Native histograms](/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/metrics-generator/#native-histograms) for endpoint and query updates.
- **Active series cap**: `max_active_series` is a per-instance limit.
  A value of `0` disables the check.
  Refer to the [configuration reference](/docs/tempo/<TEMPO_VERSION>/configuration/#metrics-generator) for the option, and [Active series limiting](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/components/metrics-generator/#active-series-limiting) for overflow-series behavior.
- **Disabled intrinsic dimensions**: turning off `span_name` and `span_kind` removes their contribution to cardinality.
- **Longer collection interval**: a larger `collection_interval` reduces remote-write volume.
  The accepted range is 15 seconds to 5 minutes.

You can reduce cardinality further with either of these alternatives:

- To drop the latency histogram entirely, enable only the `span-metrics-count` and `span-metrics-size` subprocessors instead of `span-metrics`.
- To keep the latency histogram as a classic histogram but make it smaller, reduce the number of `histogram_buckets`.
  Refer to [Configure histogram buckets](/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/span-metrics/span-metrics-metrics-generator/#configure-histogram-buckets) for how to reduce or extend the bucket range.

## Full-featured configuration

This configuration adds custom dimensions, renames an attribute to a shorter label, and applies filter policies.
Use it as a reference for combining options rather than as a default.

```yaml
# tempo.yaml
metrics_generator:
  storage:
    path: /var/tempo/generator/wal
    remote_write:
      - url: http://prometheus:9090/api/v1/write
        send_exemplars: true

overrides:
  defaults:
    metrics_generator:
      processors:
        - span-metrics
        - service-graphs
        - host-info
      processor:
        span_metrics:
          # Additional attributes to surface as labels. These only appear if
          # the attribute is present on the span. Consider cardinality before
          # adding high-cardinality attributes such as http.route.
          dimensions:
            - http.method
            - http.route
            - deployment.environment
          # Rename k8s.cluster.name to the shorter label "cluster".
          dimension_mappings:
            - name: cluster
              source_labels: ["k8s.cluster.name"]
          intrinsic_dimensions:
            status_message: true
          # Only generate metrics for server spans, and drop health-check noise.
          filter_policies:
            - include:
                match_type: strict
                attributes:
                  - key: kind
                    value: SPAN_KIND_SERVER
            - exclude:
                match_type: regex
                attributes:
                  - key: name
                    value: .*health.*
        service_graphs:
          dimensions:
            - deployment.environment
```

The `dimensions` list includes low-cardinality attributes such as `http.method` and `deployment.environment`,
and a higher-cardinality `http.route` attribute.
A dimension can only surface an attribute that already exists on your spans.
If the attribute isn't present, the generator produces no label and no error.
For the cardinality table and that warning, refer to [Adding custom dimensions](/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/span-metrics/span-metrics-metrics-generator/#adding-custom-dimensions).

The `filter_policies` block includes only server spans and excludes health-check span names.
For `include`, `include_any`, and `exclude` patterns, including filtering by service name, refer to [Filtering](/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/span-metrics/span-metrics-metrics-generator/#filtering).

The `service_graphs.dimensions` block adds `deployment.environment` to service-graph series.
For peer attributes, client and server prefixes, and other service-graph options, refer to [Service graphs](/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/service_graphs/).

The host-info processor emits a `traces_host_info` gauge with `grafana_host_id` and `host_source` labels.
It only produces series for spans whose resource carries one of the configured `host_identifiers` attributes, which default to `host.id` and `k8s.node.name`.
If your spans don't include a host identifier, the processor generates no `traces_host_info` series.
To tune host identifiers or the metric name, refer to the [configuration reference](/docs/tempo/<TEMPO_VERSION>/configuration/#metrics-generator).

For a detailed explanation of each span-metrics option, including worked examples for dimensions, `dimension_mappings`, intrinsic dimensions, histogram buckets, and filter policies, refer to [Use the metrics-generator to create metrics from spans](/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/span-metrics/span-metrics-metrics-generator/).

## Version and format notes

- **Tempo 3.0 removed the `local-blocks` processor.**
  Remove any `local-blocks` entries from `metrics_generator.processors`.
  The live-store component now serves TraceQL metrics queries on recent data.
  The valid processors are `span-metrics`, `service-graphs`, and `host-info`, along with the span-metrics and service-graphs subprocessors.
- **Filter policy validation is stricter in Tempo 3.0.**
  Attribute keys must be valid TraceQL identifiers, non-intrinsic keys must include a `resource.` or `span.` scope, and intrinsic values such as `kind` and `status` must be recognized values.
  If you're upgrading from Tempo 2.x, refer to the [3.0 release notes](/docs/tempo/<TEMPO_VERSION>/release-notes/v3-0/#stricter-filter-policy-validation).

## Apply the configuration

Top-level `metrics_generator` settings, such as `storage.remote_write` and `registry`, are loaded at startup.
Restart the metrics-generator target after you change them, or after you change static `overrides` in `tempo.yaml`.

If you manage tenant settings in a runtime overrides file or through the [user-configurable overrides](/docs/tempo/<TEMPO_VERSION>/operations/manage-advanced-systems/user-configurable-overrides/) API, Tempo reloads those settings without a restart.

## Verify the generator is producing metrics

After you apply a configuration, confirm the generator is active and writing metrics:

1. Confirm the generator is producing series.
   In Prometheus, `tempo_metrics_generator_registry_active_series` should be greater than zero for the tenant.
1. Confirm remote-write is succeeding.
   `prometheus_remote_storage_samples_failed_total` should stay flat.
   A rising value indicates a problem with the remote-write endpoint.
1. Query the generated metrics directly.
   For example, `traces_spanmetrics_calls_total` or `traces_service_graph_request_total` should return data.

If no series appear, refer to [Troubleshoot metrics-generator](/docs/tempo/<TEMPO_VERSION>/troubleshooting/metrics-generator/).

## Next steps

- [Use the metrics-generator to create metrics from spans](/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/span-metrics/span-metrics-metrics-generator/) for detailed span-metrics options.
- [Service graphs](/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/service_graphs/) for service-graph processor options and metrics.
- [Choose where to generate metrics from traces](/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/where-to-generate-metrics/) to decide between the metrics-generator and client-side generation, and to understand how sampling affects coverage.
- [Cardinality](/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/metrics-generator/cardinality/) to understand how configuration affects active series.
- [Troubleshoot metrics-generator](/docs/tempo/<TEMPO_VERSION>/troubleshooting/metrics-generator/) to diagnose missing or unexpected metrics.
