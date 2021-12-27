local dashboard_utils = import 'dashboard-utils.libsonnet';
local g = import 'grafana-builder/grafana.libsonnet';

dashboard_utils {
  grafanaDashboards+: {
    'tempo-tenants.json':
      $.dashboard('Tempo / Tenants')

      .addLogsDatasourceTemplate()
      .addClusterSelectorTemplates()
      .addQueryResultTemplate('tenant', 'sum(tempodb_blocklist_length{cluster=~"$cluster", namespace=~"$namespace"}) by (tenant) > 0')

      .addRow(
        g.row('Cluster')
        .addPanel(
          g.panel('Distributor Spans/Second (topk 10)') +
          $.queryPanel(
            |||
              topk(10,
                sum(
                  rate(tempo_distributor_spans_received_total{%s}[$__rate_interval])
                ) by (tenant)
              )
            ||| % $.jobMatcher($._config.jobs.distributor),
            '{{tenant}}',
          ),
        )
        .addPanel(
          g.panel('Ingester Traces Created/Second (topk 10)') +
          $.queryPanel(
            |||
              topk(10,
                sum(
                  rate(tempo_ingester_traces_created_total{%s}[$__rate_interval])
                ) by (tenant)
              )
            ||| % $.jobMatcher($._config.jobs.ingester),
            '{{tenant}}',
          ),
        ),
      )
      .addRow(
        g.row('Cluster')
        .addPanel(
          g.panel('Distributor Bytes/Second (topk 10)') +
          $.queryPanel(
            |||
              topk(10,
                sum(
                  rate(tempo_distributor_bytes_received_total{%s}[$__rate_interval])
                ) by (tenant)
              )
            ||| % $.jobMatcher($._config.jobs.distributor),
            '{{tenant}}',
          ),
        )
        .addPanel(
          g.panel('Blocklist Length (topk 10)') +
          $.queryPanel(
            |||
              topk(10,
                avg(
                  tempodb_blocklist_length{%s}
                ) by (tenant)
              )
            ||| % $.jobMatcher($._config.jobs.compactor),
            '{{tenant}}',
          ),
        )
      )
      .addRow(
        g.row('Tenant')
        .addPanel(
          g.panel('Distributor Spans/Second') +
          $.queryPanel(
            |||
              sum(rate(tempo_distributor_spans_received_total{job=~".*/distributor", tenant=~"$tenant"}[$__rate_interval])) by (cluster, namespace)
            |||,
            'Received ({{cluster}}, {{namespace}})',
          ) +
          $.queryPanel(
            |||
              sum(rate(tempo_discarded_spans_total{job=~".*/distributor", tenant=~"$tenant"}[$__rate_interval])) by (cluster, namespace, reason)
            |||,
            'Discarded: {{reason}} ({{cluster}}, {{namespace}})',
          ),
        )
        .addPanel(
          g.panel('Queries/Second') +
          $.queryPanel(
            |||
              sum(
                rate(tempo_query_frontend_queries_total{job=~".*/query-frontend", tenant=~"$tenant"}[$__rate_interval])
              ) by (cluster, namespace, op)
            |||,
            '{{op}} ({{cluster}}, {{namespace}})',
          ),
        )
        .addPanel(
          g.panel('Blocklist Length') +
          $.queryPanel(
            |||
              avg(tempodb_blocklist_length{tenant=~"$tenant"}) by (cluster, namespace)
            |||,
            '{{cluster}}, {{namespace}}',
          ),
        )
      )
      .addRow(
        g.row('Tenant - Queries')
        .addPanel(
          g.panel('Queries') +
          $.logsPanel(
            'Queries',
            |||
              {job=~".*/query-frontend"}
              |= "caller=handler.go"
              | logfmt
              | tenant="$tenant"
              | line_format `{{printf "%-12.12s" .duration}}  {{printf "%-6.6s" .response_size}}  {{.url}}`
            |||,
          )
        )
      ),
  },
}
