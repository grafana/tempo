local dashboard_utils = import 'dashboard-utils.libsonnet';
local g = import 'grafana-builder/grafana.libsonnet';

dashboard_utils {

  local limitQuery(limit) = |||
    max(
      max by (cluster, namespace, limit_name) (tempo_limits_overrides{%(matcher)s,user="$tenant",limit_name="%(limit_name)s"})
      or max by (cluster, namespace, limit_name) (tempo_limits_defaults{%(matcher)s,limit_name="%(limit_name)s"})
    ) by (%(limit_name)s)
  ||| % {
    limit_name: limit,
    matcher: $.jobMatcher($._config.jobs.compactor),  // TODO(mapno): use the correct job for each limit
  },

  local limit_style(alias) = {
    alias: alias,
    fill: 0,
    dashes: true,
  },

  grafanaDashboards+: {
    'tempo-tenants.json':
      $.dashboard('Tempo / Tenants')
      .addClusterSelectorTemplates()
      .addTemplate('tenant', 'tempodb_blocklist_length{%s}' % $.jobMatcher($._config.jobs.compactor), 'tenant')
      .addRow(
        g.row('Tenant info')
        .addPanel(
          $.panel('Limits') +
          $.tablePanel(
            [
              |||
                max(
                  max by (cluster, namespace, limit_name) (tempo_limits_overrides{%s,user="$tenant"})
                  or max by (cluster, namespace, limit_name) (tempo_limits_defaults{%s})
                ) by (limit_name)
              ||| % [$.jobMatcher($._config.jobs.compactor), $.jobMatcher($._config.jobs.compactor)],
            ], {}
          ),
        )
      )
      .addRow(
        g.row('Ingestion')
        .addPanel(
          $.panel('Distributor bytes/s') +
          $.queryPanel(
            [
              'sum(rate(tempo_distributor_bytes_received_total{%s,tenant="$tenant"}[$__rate_interval]))' % $.jobMatcher($._config.jobs.distributor),
              limitQuery('ingestion_rate_limit_bytes'),
              limitQuery('ingestion_burst_size_bytes'),
            ],
            [
              'received',
              'limit',
              'burst limit',
            ],
          ) + {
            seriesOverrides: [limit_style('limit'), limit_style('burst limit')],
            yaxes: g.yaxes('Bps'),
          },
        )
        .addPanel(
          $.panel('Distributor spans/s') +
          $.queryPanel(
            [
              'sum(rate(tempo_distributor_spans_received_total{%s,tenant="$tenant"}[$__rate_interval]))' % $.jobMatcher($._config.jobs.distributor),
              'sum(rate(tempo_discarded_spans_total{%s,tenant="$tenant"}[$__rate_interval])) by (reason)' % $.jobMatcher($._config.jobs.distributor),
            ],
            [
              'accepted',
              'refused {{ reason }}',
            ],
          ),
        )
        .addPanel(
          $.panel('Live traces') +
          $.queryPanel(
            [
              'max(tempo_ingester_live_traces{%s,tenant="$tenant"})' % $.jobMatcher($._config.jobs.ingester),
              limitQuery('max_global_traces_per_user'),
              limitQuery('max_local_traces_per_user'),
            ],
            [
              'live traces',
              'global limit',
              'local limit',
            ],
          ) + { seriesOverrides: [limit_style('global limit'), limit_style('local limit')] },
        )
      )
      .addRow(
        g.row('Reads')
        .addPanel(
          $.panel('Queries/s (ID lookup)') +
          $.queryPanel(
            'sum(rate(tempo_query_frontend_queries_total{%s,tenant="$tenant",op="traces"}[$__rate_interval])) by (status)' % $.jobMatcher($._config.jobs.query_frontend),
            '{{ status }}',
          ),
        )
        .addPanel(
          $.panel('Queries/s (search)') +
          $.queryPanel(
            'sum(rate(tempo_query_frontend_queries_total{%s,tenant="$tenant",op="search"}[$__rate_interval])) by (status)' % $.jobMatcher($._config.jobs.query_frontend),
            '{{ status }}',
          ),
        )
      )
      .addRow(
        g.row('Storage')
        .addPanel(
          $.panel('Blockslist length') +
          $.queryPanel(
            'avg(tempodb_blocklist_length{%s,tenant="$tenant"})' % $.jobMatcher($._config.jobs.compactor),
            'length',
          ) + { legend: { show: false } },
        )
        .addPanel(
          $.panel('Outstanding compactions') +
          $.queryPanel(
            |||
              sum(tempodb_compaction_outstanding_blocks{%s,tenant="$tenant"})
              /
              count(tempo_build_info{%s})
            ||| % [$.jobMatcher($._config.jobs.compactor), $.jobMatcher($._config.jobs.compactor)],
            'blocks',
          ) + { legend: { show: false } },
        )
      )
      .addRow(
        g.row('Metrics generator')
        .addPanel(
          $.panel('Bytes/s') +
          $.queryPanel(
            'sum(rate(tempo_metrics_generator_bytes_received_total{%s,tenant="$tenant"}[$__rate_interval]))' % $.jobMatcher($._config.jobs.metrics_generator),
            'rate',
          ) + {
            legend: { show: false },
            yaxes: g.yaxes('Bps'),
          },
        )
        .addPanel(
          $.panel('Active series') +
          $.queryPanel(
            [
              'sum(tempo_metrics_generator_registry_active_series{%s,tenant="$tenant"})' % $.jobMatcher($._config.jobs.metrics_generator),
              limitQuery('metrics_generator_max_active_series'),
            ],
            [
              '{{ tenant }}',
              'limit',
            ],
          ) + { seriesOverrides: [limit_style('limit')] },
        )
      ),

  },
}
