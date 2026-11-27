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
          $.panel('Requests / sec') +
          $.qpsPanel('tempo_request_duration_seconds_count{%s, route=~"%sapi_.*"}' % [$.jobMatcher($._config.jobs.gateway), $._config.http_api_prefix])
        )
        .addPanel(
          $.panel('Request latency') +
          $.latencyPanel('tempo_request_duration_seconds', '{%s,route=~"%sapi_.*"}' % [$.jobMatcher($._config.jobs.gateway), $._config.http_api_prefix], additional_grouping='route')
        )
      )
      .addRow(
        g.row('Query-frontend')
        .addPanel(
          $.panel('Requests / sec') +
          $.qpsPanel('tempo_request_duration_seconds_count{%s, route=~"%sapi_.*"}' % [$.jobMatcher($._config.jobs.query_frontend), $._config.http_api_prefix])
        )
        .addPanel(
          $.panel('Request latency') +
          $.latencyPanel('tempo_request_duration_seconds', '{%s,route=~"%sapi_.*"}' % [$.jobMatcher($._config.jobs.query_frontend), $._config.http_api_prefix], additional_grouping='route')
        )
      )
      .addRow(
        g.row('Querier')
        .addPanel(
          $.panel('Requests / sec') +
          $.qpsPanel('tempo_request_duration_seconds_count{%s, route=~"querier_%sapi_.*"}' % [$.jobMatcher($._config.jobs.querier), $._config.http_api_prefix])
        )
        .addPanel(
          $.panel('Request latency') +
          $.latencyPanel('tempo_request_duration_seconds', '{%s,route=~"querier_%sapi_.*"}' % [$.jobMatcher($._config.jobs.querier), $._config.http_api_prefix], additional_grouping='route')
        )
        .addPanel(
          $.panel('Requests executed') +
          $.latencyPanel('tempo_querier_worker_request_executed_total', '{%s,route=~"querier_%sapi_.*"}' % [$.jobMatcher($._config.jobs.querier), $._config.http_api_prefix], additional_grouping='route')
        )
      )
      .addRow(
        g.row('Livestore')
        .addPanel(
          $.panel('Requests / sec') +
          $.qpsPanel('tempo_request_duration_seconds_count{%s, route=~"/tempopb.Querier/.*"}' % $.containerMatcher($._config.jobs.live_store))
        )
        .addPanel(
          $.panel('Request latency') +
          $.latencyPanel('tempo_request_duration_seconds', '{%s,route=~"/tempopb.Querier/.*"}' % $.containerMatcher($._config.jobs.live_store), additional_grouping='route')
        )
        .addPanel(
          $.panel('Pending queue length') +
          $.queryPanel(
            'tempo_live_store_complete_queue_length{%s}' % $.podMatcher('live-store-zone.*'), '{{pod}}'
          ) + { fieldConfig+: { defaults+: { unit: 'short' } } }
        )
        .addPanel(
          $.panel('Block completion rate') +
          $.queryPanel('sum by(pod) (rate(tempo_live_store_blocks_completed_total{%s}[$__rate_interval]))' % $.containerMatcher($._config.jobs.live_store), '{{pod}}') +
          { fieldConfig+: { defaults+: { unit: 'ops' } } }
        )
      )
      .addRow(
        $.row('Livestore partitions')
        .addPanel(
          $.timeseriesPanel('Record lag by partition') +
          $.panelDescription(
            'Record lag by partition',
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
          $.timeseriesPanel('Time lag by partition') +
          $.panelDescription(
            'Time lag by partition',
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
          $.panel('Requests / sec') +
          $.qpsPanel('tempo_memcache_request_duration_seconds_count{%s,method=~"Memcache.Get|Memcache.GetMulti"}' % $.jobMatcher($._config.jobs.querier))
        )
        .addPanel(
          $.panel('Read latency') +
          $.latencyPanel('tempo_memcache_request_duration_seconds', '{%s,method=~"Memcache.Get|Memcache.GetMulti"}' % $.jobMatcher($._config.jobs.querier))
        )
      )
      .addRow(
        g.row('Backend')
        .addPanel(
          $.panel('Requests / sec') +
          $.qpsPanel('tempodb_backend_request_duration_seconds_count{%s,operation="GET"}' % $.jobMatcher($._config.jobs.querier))
        )
        .addPanel(
          $.panel('Read latency') +
          $.latencyPanel('tempodb_backend_request_duration_seconds', '{%s,operation="GET"}' % $.jobMatcher($._config.jobs.querier))
        )
      )
      .addRow(
        g.row('External requests')
        .addPanel(
          $.panel('Requests / sec') +
          $.qpsPanel('tempo_querier_external_endpoint_request_duration_seconds_count{%s}' % $.jobMatcher($._config.jobs.querier))
        )
        .addPanel(
          $.panel('Request latency') +
          $.latencyPanel('tempo_querier_external_endpoint_request_duration_seconds', '{%s}' % $.jobMatcher($._config.jobs.querier))
        )
      )
      // Autoscaling rows are grouped at the bottom so they don't interleave with the request-path rows.
      .addRowIf($._config.autoscaling.query_frontend.enabled, $.autoscalingRow('Query-frontend', 'query_frontend'))
      .addRowIf($._config.autoscaling.querier.enabled, $.autoscalingRow('Querier', 'querier')),
  },
}
