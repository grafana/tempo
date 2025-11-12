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
        .addPanel(
          $.panel('Requests Executed') +
          $.latencyPanel('tempo_querier_worker_request_executed_total', '{%s,route=~"querier_%sapi_.*"}' % [$.jobMatcher($._config.jobs.querier), $._config.http_api_prefix], additional_grouping='route')
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
        g.row('Livestore')
        .addPanel(
          $.panel('QPS') +
          $.qpsPanel('tempo_request_duration_seconds_count{%s, route=~"/tempopb.Querier/.*"}' % $.containerMatcher($._config.jobs.live_store))
        )
        .addPanel(
          $.panel('Latency') +
          $.latencyPanel('tempo_request_duration_seconds', '{%s,route=~"/tempopb.Querier/.*"}' % $.containerMatcher($._config.jobs.live_store), additional_grouping='route')
        )
        .addPanel(
          $.panel('Pending Queue Length') +
          $.queryPanel(
            'tempo_live_store_complete_queue_length{%s}' % $.podMatcher('live-store-zone.*'), '{{pod}}'
          )
        )
        .addPanel(
          $.panel('Completed Blocks') +
          $.queryPanel('sum by(pod) (rate(tempo_live_store_blocks_completed_total{%s}[$__rate_interval]))' % $.containerMatcher($._config.jobs.live_store), '{{pod}}')
        )
      )
      .addRow(
        $.row('Livestore Partitions')
        .addPanel(
          $.timeseriesPanel('Lag of records by partition') +
          $.panelDescription(
            'Kafka lag by partition records',
            'Overview of the lag by partition in records.',
          ) +
          $.queryPanel(
            'max(tempo_ingest_group_partition_lag{%(job)s}) by (partition,group)' % { job: $.jobMatcher('live-store-zone.*') }, '{{partition}}'
          ) +
          $.stack
          +
          { fieldConfig+: { defaults+: { unit: 'short' } } },
        )
        .addPanel(
          $.timeseriesPanel('Lag by partition (sec)') +
          $.panelDescription(
            'Kafka lag by partition in seconds',
            'Overview of the lag by partition in seconds.',
          ) +
          $.queryPanel(
            'max(tempo_ingest_group_partition_lag_seconds{%(job)s}) by (partition,group)' % { job: $.jobMatcher('live-store-zone.*') }, '{{partition}}'
          ) +
          $.stack
          +
          { fieldConfig+: { defaults+: { unit: 's' } } },
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
