local dashboard_utils = import 'dashboard-utils.libsonnet';
local g = import 'grafana-builder/grafana.libsonnet';

dashboard_utils {
  grafanaDashboards+: {
    'tempo-reads.json':
      $.dashboard('Tempo / Reads')
      .addClusterSelectorTemplates()
      .addRow(
        g.row('Gateway')
        .addPanel(
          $.panel('QPS') +
          $.qpsPanel('tempo_request_duration_seconds_count{%s, route=~"%sapi_.*"}' % [$.jobMatcher($._config.jobs.gateway), $._config.http_api_prefix])
        )
        .addPanel(
          $.panel('Latency') +
          $.latencyPanel('tempo_request_duration_seconds', '{%s,route=~"%sapi_.*"}' % [$.jobMatcher($._config.jobs.gateway), $._config.http_api_prefix], additional_grouping='route')
        )
      )
      .addRow(
        g.row('Query Frontend')
        .addPanel(
          $.panel('QPS') +
          $.qpsPanel('tempo_request_duration_seconds_count{%s, route=~"%sapi_.*"}' % [$.jobMatcher($._config.jobs.query_frontend), $._config.http_api_prefix])
        )
        .addPanel(
          $.panel('Latency') +
          $.latencyPanel('tempo_request_duration_seconds', '{%s,route=~"%sapi_.*"}' % [$.jobMatcher($._config.jobs.query_frontend), $._config.http_api_prefix], additional_grouping='route')
        )
      )
      .addRow(
        g.row('Querier')
        .addPanel(
          $.panel('QPS') +
          $.qpsPanel('tempo_request_duration_seconds_count{%s, route=~"querier_%sapi_.*"}' % [$.jobMatcher($._config.jobs.querier), $._config.http_api_prefix])
        )
        .addPanel(
          $.panel('Latency') +
          $.latencyPanel('tempo_request_duration_seconds', '{%s,route=~"querier_%sapi_.*"}' % [$.jobMatcher($._config.jobs.querier), $._config.http_api_prefix], additional_grouping='route')
        )
      )
      .addRow(
        g.row('Ingester')
        .addPanel(
          $.panel('QPS') +
          $.qpsPanel('tempo_request_duration_seconds_count{%s, route=~"/tempopb.Querier/.*"}' % $.jobMatcher($._config.jobs.ingester))
        )
        .addPanel(
          $.panel('Latency') +
          $.latencyPanel('tempo_request_duration_seconds', '{%s,route=~"/tempopb.Querier/.*"}' % $.jobMatcher($._config.jobs.ingester), additional_grouping='route')
        )
      )
      .addRow(
        g.row('Memcached')
        .addPanel(
          $.panel('QPS') +
          $.qpsPanel('cortex_memcache_request_duration_seconds_count{%s,method=~"Memcache.Get|Memcache.GetMulti"}' % $.jobMatcher($._config.jobs.querier))
        )
        .addPanel(
          $.panel('Latency') +
          $.latencyPanel('cortex_memcache_request_duration_seconds', '{%s,method=~"Memcache.Get|Memcache.GetMulti"}' % $.jobMatcher($._config.jobs.querier))
        )
      )
      .addRow(
        g.row('Backend')
        .addPanel(
          $.panel('QPS') +
          $.qpsPanel('tempodb_backend_request_duration_seconds_count{%s,operation="GET"}' % $.jobMatcher($._config.jobs.querier))
        )
        .addPanel(
          $.panel('Latency') +
          $.latencyPanel('tempodb_backend_request_duration_seconds', '{%s,operation="GET"}' % $.jobMatcher($._config.jobs.querier))
        )
      )
      .addRow(
        g.row('TempoDB Access')
        .addPanel(
          g.panel('p99') +
          g.queryPanel('histogram_quantile(.99, sum(rate(tempo_query_reads_bucket[$__interval])) by (layer, le))', '{{layer}}'),
        )
        .addPanel(
          g.panel('p50') +
          g.queryPanel('histogram_quantile(.5, sum(rate(tempo_query_reads_bucket[$__interval])) by (layer, le))', '{{layer}}'),
        )
        .addPanel(
          g.panel('Bytes Read') +
          g.queryPanel('histogram_quantile(.99, sum(rate(tempo_query_bytes_read_bucket[$__interval])) by (le))', '0.99') +
          g.queryPanel('histogram_quantile(.9, sum(rate(tempo_query_bytes_read_bucket[$__interval])) by (le))', '0.9') +
          g.queryPanel('histogram_quantile(.5, sum(rate(tempo_query_bytes_read_bucket[$__interval])) by (le))', '0.5'),
        )
      ),
  },
}
