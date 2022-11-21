local g = import 'grafana-builder/grafana.libsonnet';

local row_settings = {
  height: '100px',
  showTitle: false,
};

local panel_settings = {
  repeat: 'instance',
  colorBackground: true,
  thresholds: '0.5,0.5',
};

{
  grafanaDashboards+:: {
    'consul-overview.json':
      g.dashboard('Consul Overview', std.md5('20210205-consul'))
      .addTemplate('job', 'consul_up', 'job')
      .addMultiTemplate('instance', 'consul_up{job="$job"}', 'instance')
      .addRow(
        g.row('Up')
        .addPanel(
          g.panel('$instance') +
          g.statPanel('consul_up{job="$job",instance=~"$instance"}', 'none') +
          panel_settings {
            valueMaps: [
              { value: '0', op: '=', text: 'DOWN' },
              { value: '1', op: '=', text: 'UP' },
            ],
            colors: ['#d44a3a', 'rgba(237, 129, 40, 0.89)', '#299c46'],
          }
        ) +
        row_settings
      )
      .addRow(
        g.row('Leader')
        .addPanel(
          g.panel('$instance') +
          g.statPanel(|||
            (rate(consul_raft_leader_lastcontact_count{job="$job",instance=~"$instance"}[$__rate_interval]) > bool 0)
              or
            (consul_up{job="$job",instance=~"$instance"} == bool 0)
          |||, 'none') +
          panel_settings {
            valueMaps: [
              { value: '0', op: '=', text: 'FOLLOWER' },
              { value: '1', op: '=', text: 'LEADER' },
            ],
            colors: ['rgba(237, 129, 40, 0.89)', 'rgba(237, 129, 40, 0.89)', '#299c46'],
          }
        ) +
        row_settings
      )
      .addRow(
        g.row('Has Leader')
        .addPanel(
          g.panel('$instance') +
          g.statPanel('consul_raft_leader{job="$job",instance=~"$instance"}', 'none') +
          panel_settings {
            valueMaps: [
              { value: '0', op: '=', text: 'NO LEADER' },
              { value: '1', op: '=', text: 'HAS LEADER' },
            ],
            colors: ['#d44a3a', 'rgba(237, 129, 40, 0.89)', '#299c46'],
          }
        ) +
        row_settings
      )
      .addRow(
        g.row('# Peers')
        .addPanel(
          g.panel('$instance') +
          g.statPanel('consul_raft_peers{job="$job",instance=~"$instance"}', 'none') +
          panel_settings {
            thresholds: '1,2',
            colors: ['#d44a3a', 'rgba(237, 129, 40, 0.89)', '#299c46'],
          }
        ) +
        row_settings
      )
      .addRow(
        g.row('Consul Server')
        .addPanel(
          g.panel('QPS') +
          g.queryPanel('sum(rate(consul_http_request_count{job="$job"}[$__rate_interval])) by (instance)', '{{instance}}') +
          g.stack
        )
        .addPanel(
          g.panel('Latency') +
          g.queryPanel('max(consul_http_request{job="$job", quantile="0.99"})', '99th Percentile') +
          g.queryPanel('max(consul_http_request{job="$job", quantile="0.5"})', '50th Percentile') +
          g.queryPanel('sum(rate(consul_http_request{job="$job"}[5m])) / sum(rate(consul_http_request{job="$job"}[5m]))', 'Average') +
          { yaxes: g.yaxes('ms') }
        )
      ),
  },
}
