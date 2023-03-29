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
        g.row('Querier External Endpoint')
        .addPanel(
          $.panel('QPS') +
          $.qpsPanel('tempo_querier_external_endpoint_duration_seconds_count{%s}' % [$.jobMatcher($._config.jobs.querier)])
        )
        .addPanel(
          $.panel('Latency') +
          $.latencyPanel('tempo_querier_external_endpoint_duration_seconds', '{%s}' % [$.jobMatcher($._config.jobs.querier)], additional_grouping='endpoint')
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
          $.qpsPanel('tempo_memcache_request_duration_seconds_count{%s,method=~"Memcache.Get|Memcache.GetMulti"}' % $.jobMatcher($._config.jobs.querier))
        )
        .addPanel(
          $.panel('Latency') +
          $.latencyPanel('tempo_memcache_request_duration_seconds', '{%s,method=~"Memcache.Get|Memcache.GetMulti"}' % $.jobMatcher($._config.jobs.querier))
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
      ),
  },
}
