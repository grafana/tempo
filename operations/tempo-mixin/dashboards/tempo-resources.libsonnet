local dashboard_utils = import 'dashboard-utils.libsonnet';
local g = import 'grafana-builder/grafana.libsonnet';

dashboard_utils {
  local cpuPanel(job) =
    $.containerCPUUsagePanel('CPU', job) +
    { fieldConfig+: { defaults+: { unit: 'cores' } } },
  local memoryPanel(job) =
    $.containerMemoryWorkingSetPanel('Memory (workingset)', job) +
    { fieldConfig+: { defaults+: { unit: 'bytes' } } },
  local heapPanel(title, job) =
    $.goHeapInUsePanel(title, job) +
    { fieldConfig+: { defaults+: { unit: 'bytes' } } },

  grafanaDashboards+: {
    'tempo-resources.json':
      $.dashboard('Tempo / Resources')
      .addClusterSelectorTemplates()
      .addRow(
        g.row('Gateway')
        .addPanel(
          cpuPanel($._config.jobs.gateway),
        )
        .addPanel(
          memoryPanel($._config.jobs.gateway),
        )
        .addPanel(
          heapPanel('Memory (go heap inuse)', $.jobMatcher($._config.jobs.gateway)),
        )
      )
      .addRow(
        g.row('Distributor')
        .addPanel(
          cpuPanel($._config.jobs.distributor),
        )
        .addPanel(
          memoryPanel($._config.jobs.distributor),
        )
        .addPanel(
          heapPanel('Memory (go heap inuse)', $.jobMatcher($._config.jobs.distributor)),
        )
      )
      .addRow(
        g.row('Livestore')
        .addPanel(
          cpuPanel($._config.jobs.live_store),
        )
        .addPanel(
          memoryPanel($._config.jobs.live_store),
        )
        .addPanel(
          heapPanel('Memory (go heap inuse)', $.containerMatcher($._config.jobs.live_store)),
        )
      )
      .addRow(
        g.row('Metrics-generator')
        .addPanel(
          cpuPanel($._config.jobs.metrics_generator),
        )
        .addPanel(
          memoryPanel($._config.jobs.metrics_generator),
        )
        .addPanel(
          heapPanel('Memory (go heap inuse)', $.jobMatcher($._config.jobs.metrics_generator)),
        )
      )
      .addRow(
        g.row('Query Frontend')
        .addPanel(
          cpuPanel($._config.jobs.query_frontend),
        )
        .addPanel(
          memoryPanel($._config.jobs.query_frontend),
        )
        .addPanel(
          heapPanel('Memory (go heap inuse)', $.jobMatcher($._config.jobs.query_frontend)),
        )
      )
      .addRow(
        g.row('Querier')
        .addPanel(
          cpuPanel($._config.jobs.querier),
        )
        .addPanel(
          memoryPanel($._config.jobs.querier),
        )
        .addPanel(
          heapPanel('Memory (go heap inuse)', $.jobMatcher($._config.jobs.querier)),
        )
      )
      .addRow(
        g.row('Compactor')
        .addPanel(
          cpuPanel($._config.jobs.compactor),
        )
        .addPanel(
          memoryPanel($._config.jobs.compactor),
        )
        .addPanel(
          heapPanel('Memory (go heap inuse)', $.jobMatcher($._config.jobs.compactor)),
        )
      )
      .addRow(
        g.row('Memcached')
        .addPanel(
          cpuPanel($._config.jobs.memcached),
        )
        .addPanel(
          memoryPanel($._config.jobs.memcached),
        )
      )

      .addRow(
        g.row('Backend scheduler')
        .addPanel(
          cpuPanel($._config.jobs.backend_scheduler),
        )
        .addPanel(
          memoryPanel($._config.jobs.backend_scheduler),
        )
        .addPanel(
          heapPanel('Memory (go heap inuse)', $.jobMatcher($._config.jobs.backend_scheduler)),
        )
      )
      .addRow(
        g.row('Backend worker')
        .addPanel(
          cpuPanel($._config.jobs.backend_worker),
        )
        .addPanel(
          memoryPanel($._config.jobs.backend_worker),
        )
        .addPanel(
          heapPanel('Memory (go heap inuse)', $.jobMatcher($._config.jobs.backend_worker)),
        )
      )
      .addRow(
        g.row('Block builder')
        .addPanel(
          cpuPanel($._config.jobs.block_builder),
        )
        .addPanel(
          memoryPanel($._config.jobs.block_builder),
        )
        .addPanel(
          heapPanel('Memory (go heap inuse)', $.jobMatcher($._config.jobs.block_builder)),
        )
      )
      .addRow(
        g.row('Live store')
        .addPanel(
          cpuPanel($._config.jobs.live_store),
        )
        .addPanel(
          memoryPanel($._config.jobs.live_store),
        )
        .addPanel(
          heapPanel('Memory (go heap inuse)', $.jobMatcher('live-store-.*')),
        )
      ),
  },
}
