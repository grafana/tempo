local utils = import '../lib/utils.libsonnet';
local g = import 'grafana-builder/grafana.libsonnet';
local grafana = import 'grafonnet/grafana.libsonnet';

local dashboard = grafana.dashboard;
local prometheus = grafana.prometheus;
local graphPanel = grafana.graphPanel;
local row = grafana.row;

local host_matcher = 'job=~"$job", instance=~"$instance"';

// Templates
local ds_template = {
  current: {
    text: 'default',
    value: 'default',
  },
  hide: 0,
  label: 'Data Source',
  name: 'prometheus_datasource',
  options: [],
  query: 'prometheus',
  refresh: 1,
  regex: '',
  type: 'datasource',
};

local job_template = grafana.template.new(
  'job',
  '$prometheus_datasource',
  'label_values(agent_build_info, job)',
  label='job',
  refresh='load',
  multi=true,
  includeAll=true,
  sort=1,
);

local instance_template = grafana.template.new(
  'instance',
  '$prometheus_datasource',
  'label_values(agent_build_info{job=~"$job"}, instance)',
  label='instance',
  refresh='load',
  multi=true,
  includeAll=true,
  sort=1,
);

{
  grafanaDashboards+:: {
    'grafana-agent-operational.json':
      local garbageCollectionSeconds =
        graphPanel.new(
          'Garbage Collection Seconds',
          description='A summary of the pause duration of garbage collection cycles.',
          datasource='$prometheus_datasource',
          span=6,
        )
        .addTarget(prometheus.target(
          'rate(go_gc_duration_seconds_count{' + host_matcher + '}[$__rate_interval])',
          legendFormat='{{instance}}'
        )) +
        utils.timeSeriesOverride(unit='s');

      local goHeap =
        graphPanel.new(
          'Go Heap',
          description='Number of heap bytes that are in use.',
          datasource='$prometheus_datasource',
          span=6,
        )
        .addTarget(prometheus.target(
          'go_memstats_heap_inuse_bytes{' + host_matcher + '}',
          legendFormat='{{instance}}'
        )) +
        utils.timeSeriesOverride(unit='decbytes');

      local goRoutines =
        graphPanel.new(
          'Go Routines',
          description='Number of goroutines that currently exist.',
          datasource='$prometheus_datasource',
          span=6,
        )
        .addTarget(prometheus.target(
          'go_goroutines{' + host_matcher + '}',
          legendFormat='{{instance}}'
        )) +
        utils.timeSeriesOverride(unit='short');

      local cpuUsage =
        graphPanel.new(
          'CPU Usage',
          description='Total user and system CPU time spent in seconds.',
          datasource='$prometheus_datasource',
          span=6,
        )
        .addTarget(prometheus.target(
          'rate(process_cpu_seconds_total{' + host_matcher + '}[$__rate_interval])',
          legendFormat='{{instance}}',
        )) +
        utils.timeSeriesOverride(unit='percent');

      local TCPConnections =
        graphPanel.new(
          'TCP Connections',
          description='Number of accepted TCP connections.',
          datasource='$prometheus_datasource',
          span=6,
        )
        .addTarget(prometheus.target(
          'agent_tcp_connections{' + host_matcher + '}',
          legendFormat='{{protocol}}'
        )) +
        utils.timeSeriesOverride(unit='short');

      local bytesSeriesPod =
        graphPanel.new(
          'Bytes/Series/Instance',
          description='Average bytes over number of active series being tracked by the WAL storage grouped by instance.',
          datasource='$prometheus_datasource',
          span=6,
        )
        .addTarget(prometheus.target(
          '\n            (sum by (instance) (avg_over_time(go_memstats_heap_inuse_bytes{' + host_matcher + '}[$__rate_interval])))\n            /\n            (sum by (instance) (agent_wal_storage_active_series{' + host_matcher + '}))\n          ',
          legendFormat='{{instance}}'
        )) +
        utils.timeSeriesOverride(unit='decbytes');

      local bytesSeries =
        graphPanel.new(
          'Bytes/Series/Job',
          description='Average bytes over number of active series being tracked by the WAL storage by Job.',
          datasource='$prometheus_datasource',
          span=6,
        )
        .addTarget(prometheus.target(
          '\n            (sum by (job) (avg_over_time(go_memstats_heap_inuse_bytes{' + host_matcher + '}[$__rate_interval])))\n            /\n            (sum by (job) (agent_wal_storage_active_series{' + host_matcher + '}))\n          ',
          legendFormat='{{job}}'
        )) +
        utils.timeSeriesOverride(unit='decbytes');

      local seriesPod =
        graphPanel.new(
          'Series/Instance',
          description='Number of active series being tracked by the WAL storage grouped by instance.',
          datasource='$prometheus_datasource',
          span=6,
        )
        .addTarget(prometheus.target(
          'sum by (instance) (agent_wal_storage_active_series{' + host_matcher + '})',
          legendFormat='{{instance}}'
        )) +
        utils.timeSeriesOverride(unit='short');

      local seriesConfig =
        graphPanel.new(
          'Series/Config/Instance',
          description='Number of active series being tracked by the WAL storage grouped by instance.',
          datasource='$prometheus_datasource',
          span=6,
        )
        .addTarget(prometheus.target(
          'sum by (instance) (agent_wal_storage_active_series{' + host_matcher + '})',
          legendFormat='{{instance}}'
        )) +
        utils.timeSeriesOverride(unit='short');

      local seriesTotal =
        graphPanel.new(
          'Series/Config/Job',
          description='Number of active series being tracked by the WAL storage grouped by job.',
          datasource='$prometheus_datasource',
          span=6,
        )
        .addTarget(prometheus.target(
          'sum by (job) (agent_wal_storage_active_series{' + host_matcher + '})',
          legendFormat='{{job}}'
        )) +
        utils.timeSeriesOverride(unit='short');

      dashboard.new('Grafana Agent Operational', tags=$._config.dashboardTags, editable=false, time_from='%s' % $._config.dashboardPeriod, uid='integration-agent-opr')
      .addTemplates([
        ds_template,
        job_template,
        instance_template,
      ])
      .addLink(grafana.link.dashboards(
        asDropdown=false,
        title='Grafana Agent Dashboards',
        includeVars=true,
        keepTime=true,
        tags=($._config.dashboardTags),
      ))
      .addRow(
        row.new('General')
        .addPanel(garbageCollectionSeconds)
        .addPanel(goHeap)
        .addPanel(goRoutines)
        .addPanel(cpuUsage)
      )
      .addRow(
        row.new('Network')
        .addPanel(TCPConnections)
      )
      .addRow(
        row.new('Prometheus Read')
        .addPanel(bytesSeriesPod)
        .addPanel(bytesSeries)
        .addPanel(seriesPod)
        .addPanel(seriesConfig)
        .addPanel(seriesTotal)
      ),
  },
}
