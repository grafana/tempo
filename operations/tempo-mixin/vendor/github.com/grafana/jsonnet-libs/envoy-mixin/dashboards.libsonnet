local g = import 'grafana-builder/grafana.libsonnet';
local template = import 'grafonnet/template.libsonnet';

{
  // Manually define the "instance" variable template in order to be able to change the "refresh" setting
  // and customise the all value.
  local instanceTemplate =
    template.new(
      name='instance',
      datasource='$datasource',
      query='label_values(envoy_server_uptime{job="$job"}, instance)',
      label='Data Source',
      allValues='.*',  // Make sure to always include all instances when "All" is selected.
      current='',
      hide='',
      refresh=2,  // Refresh on time range change.
      includeAll=true,
      sort=1
    ),

  // Envoy metrics:
  // - HTTP: https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/router_filter#statistics
  grafanaDashboards+:: {
    'envoy-overview.json':
      g.dashboard('Envoy Overview', std.md5('20210205-envoy'))
      .addTemplate('job', 'envoy_server_uptime', 'job')

      // Hidden variables to be able to repeat panels for each upstream/downstream.
      .addMultiTemplate('envoy_cluster', 'envoy_cluster_version{job=~"$job",instance=~"$instance",envoy_cluster_name!="envoy-admin"}', 'envoy_cluster_name', 2)
      .addMultiTemplate('envoy_listener_filter', 'envoy_http_downstream_rq_total{job=~"$job",instance=~"$instance",envoy_http_conn_manager_prefix!~"admin|metrics",}', 'envoy_http_conn_manager_prefix', 2)

      .addRow(
        g.row('Traffic')
        .addPanel(
          g.panel('Connections / sec') +
          g.queryPanel('sum(rate(envoy_listener_downstream_cx_total{job=~"$job",instance=~"$instance"}[$__interval]))', 'Downstream / Ingress') +
          g.queryPanel('sum(rate(envoy_cluster_upstream_cx_total{job=~"$job",instance=~"$instance"}[$__interval]))', 'Upstream / Egress') +
          { yaxes: g.yaxes('cps') }
        )
        .addPanel(
          g.panel('QPS') +
          g.queryPanel('sum(rate(envoy_http_downstream_rq_total{job=~"$job",instance=~"$instance"}[$__interval]))', 'Downstream / Ingress') +
          g.queryPanel('sum(rate(envoy_cluster_upstream_rq_total{job=~"$job",instance=~"$instance"}[$__interval]))', 'Upstream / Egress') +
          { yaxes: g.yaxes('rps') }
        )
      )

      .addRow(
        g.row('Upstream / Egress: $envoy_cluster')
        .addPanel(
          g.panel('QPS') +
          $.envoyQpsPanel('envoy_cluster_upstream_rq_xx{envoy_cluster_name="$envoy_cluster",job=~"$job",instance=~"$instance"}')
        )
        .addPanel(
          g.panel('Latency') +
          // This metric is in ms, so we apply a multiplier=1
          g.latencyPanel('envoy_cluster_upstream_rq_time', '{envoy_cluster_name="$envoy_cluster",job=~"$job",instance=~"$instance"}', '1')
        )
        .addPanel(
          g.panel('Timeouts / sec') +
          g.queryPanel('sum(rate(envoy_cluster_upstream_rq_timeout{envoy_cluster_name="$envoy_cluster",job=~"$job",instance=~"$instance"}[$__interval]))', 'Timeouts') +
          { yaxes: g.yaxes('rps') }
        )
        .addPanel(
          g.panel('Active') +
          g.queryPanel('sum(envoy_cluster_upstream_rq_active{envoy_cluster_name="$envoy_cluster",job=~"$job",instance=~"$instance"})', 'Requests') +
          g.queryPanel('sum(envoy_cluster_upstream_cx_active{envoy_cluster_name="$envoy_cluster",job=~"$job",instance=~"$instance"})', 'Connections')
        ) +

        // Repeat this row for each Envoy upstream cluster.
        { repeat: 'envoy_cluster' },
      )

      .addRow(
        g.row('Downstream / Ingress: $envoy_listener_filter')
        .addPanel(
          g.panel('QPS') +
          $.envoyQpsPanel('envoy_http_downstream_rq_xx{envoy_http_conn_manager_prefix="$envoy_listener_filter",job=~"$job",instance=~"$instance"}')
        )
        .addPanel(
          g.panel('Latency') +
          // This metric is in ms, so we apply a multiplier=1
          g.latencyPanel('envoy_http_downstream_rq_time', '{envoy_http_conn_manager_prefix="$envoy_listener_filter",job=~"$job",instance=~"$instance"}', '1')
        )
        .addPanel(
          g.panel('Timeouts / sec') +
          g.queryPanel('sum(rate(envoy_http_downstream_rq_timeout{envoy_http_conn_manager_prefix="$envoy_listener_filter",job=~"$job",instance=~"$instance"}[$__interval]))', 'Timeouts') +
          { yaxes: g.yaxes('rps') }
        )
        .addPanel(
          g.panel('Active') +
          g.queryPanel('sum(envoy_http_downstream_rq_active{envoy_http_conn_manager_prefix="$envoy_listener_filter",job=~"$job",instance=~"$instance"})', 'Requests') +
          g.queryPanel('sum(envoy_http_downstream_cx_active{envoy_http_conn_manager_prefix="$envoy_listener_filter",job=~"$job",instance=~"$instance"})', 'Connections')
        ) +

        // Repeat this row for each Envoy downstream filter.
        { repeat: 'envoy_listener_filter' },
      ) + {
        templating+: {
          list+: [instanceTemplate],
        },
      },
  },

  // This is a custom function used to display QPS by response status class captured
  // through the Envoy label "envoy_response_code_class".
  envoyQpsPanel(selector):: {
    aliasColors: {
      '1xx': '#EAB839',
      '2xx': '#7EB26D',
      '3xx': '#6ED0E0',
      '4xx': '#EF843C',
      '5xx': '#E24D42',
    },
    targets: [
      {
        expr: 'sum by (status) (label_replace(rate(' + selector + '[$__interval]), "status", "${1}xx", "envoy_response_code_class", "(.*)"))',
        format: 'time_series',
        intervalFactor: 2,
        legendFormat: '{{status}}',
        refId: 'A',
        step: 10,
      },
    ],
  } + g.stack,
}
