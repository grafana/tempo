local g = import 'grafana-builder/grafana.libsonnet';
local utils = import 'mixin-utils/utils.libsonnet';

{
  grafanaDashboards+: {
    // 'tempo-operational.json': import './tempo-operational.json',
    'tempo-reads.json':
      g.dashboard('Tempo / Reads')
      .addMultiTemplate('namespace', 'container_cpu_usage_seconds_total{}', 'namespace')
      .addMultiTemplate('component', 'container_cpu_usage_seconds_total{}', 'component')
      .addRow(
        g.row('Latency')
        .addPanel(
          g.panel('p99') +
          g.queryPanel('histogram_quantile(.99, sum(rate(tempo_query_reads_bucket[$__interval])) by (layer, le))', '{{layer}}'),
        )
        .addPanel(
          g.panel('p95') +
          g.queryPanel('histogram_quantile(.95, sum(rate(tempo_query_reads_bucket[$__interval])) by (layer, le))', '{{layer}}'),
        )
      )
      .addRow(
        g.row('Query')
        .addPanel(
          g.panel('Success Rate') +
          g.queryPanel('tempo_request_duration_seconds_sum{route="/tempopb.Querier/FindTraceByID", status_code="success"} / tempo_request_duration_seconds_sum{route="/tempopb.Querier/FindTraceByID"}', ''),
        )
        .addPanel(
          g.panel('Latency (Querier)') +
          g.queryPanel('histogram_quantile(.99, sum(rate(tempo_request_duration_seconds_bucket{method="GET", route="api_traces_traceid"}[$__interval])) by (le))', '0.99') +
          g.queryPanel('histogram_quantile(.9, sum(rate(tempo_request_duration_seconds_bucket{method="GET", route="api_traces_traceid"}[$__interval])) by (le))', '0.9') +
          g.queryPanel('histogram_quantile(.5, sum(rate(tempo_request_duration_seconds_bucket{method="GET", route="api_traces_traceid"}[$__interval])) by (le))', '0.5'),
        )
        .addPanel(
          g.panel('Latency (Ingester)') +
          g.queryPanel('histogram_quantile(.99, sum(rate(tempo_request_duration_seconds_bucket{route="/tempopb.Querier/FindTraceByID"}[$__interval])) by (le))', '0.99') +
          g.queryPanel('histogram_quantile(.9, sum(rate(tempo_request_duration_seconds_bucket{route="/tempopb.Querier/FindTraceByID"}[$__interval])) by (le))', '0.9') +
          g.queryPanel('histogram_quantile(.5, sum(rate(tempo_request_duration_seconds_bucket{route="/tempopb.Querier/FindTraceByID"}[$__interval])) by (le))', '0.5'),
        )
        .addPanel(
          g.panel('Bytes Read') +
          g.queryPanel('histogram_quantile(.99, sum(rate(tempo_query_bytes_read_bucket[$__interval])) by (le))', '0.99') +
          g.queryPanel('histogram_quantile(.9, sum(rate(tempo_query_bytes_read_bucket[$__interval])) by (le))', '0.9') +
          g.queryPanel('histogram_quantile(.5, sum(rate(tempo_query_bytes_read_bucket[$__interval])) by (le))', '0.5'),
        )
      )
      .addRow(
        g.row('Cache')
        .addPanel(
          g.panel('Lookups') +
          g.queryPanel('rate(tempodb_disk_cache_total[$__interval])', 'lookups'),
        )
        .addPanel(
          g.panel('Misses') +
          g.queryPanel('rate(tempodb_disk_cache_miss_total[$__interval])', 'misses'),
        )
        .addPanel(
          g.panel('Purges') +
          g.queryPanel('rate(tempodb_disk_cache_clean_total[$__interval])', 'purges'),
        )
      ),
  },
}
