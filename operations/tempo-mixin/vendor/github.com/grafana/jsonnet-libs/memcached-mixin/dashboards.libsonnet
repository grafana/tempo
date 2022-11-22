local g = (import 'grafana-builder/grafana.libsonnet');

{
  grafanaDashboards+: {
    'memcached-overview.json':
      (
        g.dashboard('Memcached Overview') +
        { uid: '124d5222454213f748dbfaf69b77ec48' }
      )
      .addMultiTemplate('cluster', 'memcached_commands_total', 'cluster')
      .addMultiTemplate('job', 'memcached_commands_total{cluster=~"$cluster"}', 'job')
      .addMultiTemplate('instance', 'memcached_commands_total{cluster=~"$cluster",job=~"$job"}', 'instance')
      .addRow(
        g.row('Hits')
        .addPanel(
          g.panel('Hit Rate') +
          g.queryPanel('sum(rate(memcached_commands_total{cluster=~"$cluster", job=~"$job", instance=~"$instance", command="get", status="hit"}[$__rate_interval])) / sum(rate(memcached_commands_total{cluster=~"$cluster", job=~"$job", command="get"}[$__rate_interval]))', 'Hit Rate') +
          { yaxes: g.yaxes('percentunit') },
        )
        .addPanel(
          g.panel('Top 20 Highest Connection Usage') +
          g.queryPanel(|||
            topk(20,
              max by (cluster, job, instance) (
                memcached_current_connections{cluster=~"$cluster", job=~"$job", instance=~"$instance"} / memcached_max_connections{cluster=~"$cluster", job=~"$job", instance=~"$instance"}
            ))
          |||, '{{cluster }} / {{ job }} / {{ instance }}') +
          { yaxes: g.yaxes('percentunit') },
        )
      )
      .addRow(
        g.row('Ops')
        .addPanel(
          g.panel('Commands') +
          g.queryPanel('sum by(command, status) (rate(memcached_commands_total{cluster=~"$cluster", job=~"$job", instance=~"$instance"}[$__rate_interval]))', '{{command}} {{status}}')
        )
        .addPanel(
          g.panel('Evictions') +
          g.queryPanel('sum by(instance) (rate(memcached_items_evicted_total{cluster=~"$cluster", job=~"$job", instance=~"$instance"}[$__rate_interval]))', '{{instance}}')
        )
        .addPanel(
          g.panel('Stored') +
          g.queryPanel('sum by(instance) (rate(memcached_items_total{cluster=~"$cluster", job=~"$job", instance=~"$instance"}[$__rate_interval]))', '{{instance}}')
        )
      )
      .addRow(
        g.row('Memory')
        .addPanel(
          g.panel('Memory') +
          g.queryPanel('sum by(instance) (memcached_current_bytes{cluster=~"$cluster", job=~"$job", instance=~"$instance"})', '{{instance}}') +
          g.stack +
          { yaxes: g.yaxes('bytes') },
          // TODO add memcached_limit_bytes
        )
        .addPanel(
          g.panel('Items') +
          g.queryPanel('sum by(instance) (memcached_current_items{cluster=~"$cluster", job=~"$job", instance=~"$instance"})', '{{instance}}') +
          g.stack,
        )
      )
      .addRow(
        g.row('Network')
        .addPanel(
          g.panel('Current Connections') +
          g.queryPanel([
            'sum by(instance) (memcached_current_connections{cluster=~"$cluster", job=~"$job", instance=~"$instance"})',
            // Be conservative showing the lowest setting for max connections among all selected instances.
            'min(memcached_max_connections{cluster=~"$cluster", job=~"$job", instance=~"$instance"})',
          ], [
            '{{instance}}',
            'Max Connections (min setting across all instances)',
          ])
        )
        .addPanel(
          g.panel('Connections / sec') +
          g.queryPanel([
            'sum by(instance) (rate(memcached_connections_total{cluster=~"$cluster", job=~"$job", instance=~"$instance"}[$__rate_interval]))',
          ], [
            '{{instance}}',
          ])
        )
        .addPanel(
          g.panel('Reads') +
          g.queryPanel('sum by(instance) (rate(memcached_read_bytes_total{cluster=~"$cluster", job=~"$job", instance=~"$instance"}[$__rate_interval]))', '{{instance}}') +
          { yaxes: g.yaxes('bps') },
        )
        .addPanel(
          g.panel('Writes') +
          g.queryPanel('sum by(instance) (rate(memcached_written_bytes_total{cluster=~"$cluster", job=~"$job", instance=~"$instance"}[$__rate_interval]))', '{{instance}}') +
          { yaxes: g.yaxes('bps') },
        )
      )
      .addRow(
        g.row('Memcached Info')
        .addPanel(
          g.panel('Memcached Info') +
          g.tablePanel([
            'count by (job, instance, version) (memcached_version{cluster=~"$cluster", job=~"$job", instance=~"$instance"})',
            'max by (job, instance) (memcached_uptime_seconds{cluster=~"$cluster", job=~"$job", instance=~"$instance"})',
          ], {
            job: { alias: 'Job' },
            instance: { alias: 'Instance' },
            version: { alias: 'Version' },
            'Value #A': { alias: 'Count', type: 'hidden' },
            'Value #B': { alias: 'Uptime', type: 'number', unit: 'dtdurations' },
          })
        )
      ),
  },
}
