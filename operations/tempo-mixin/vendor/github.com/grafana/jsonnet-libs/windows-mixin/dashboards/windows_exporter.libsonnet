local win = import './wintable.libsonnet';
local grafana = import 'grafonnet/grafana.libsonnet';
local prometheus = grafana.prometheus;
local graphPanel = grafana.graphPanel;

local host_matcher = 'job=~"$job", agent_hostname=~"$hostname"';

// Templates
local ds_template = {
  current: {
    text: 'default',
    value: 'default',
  },
  hide: 0,
  label: 'Data source',
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
  'label_values(windows_cs_hostname, job)',
  label='Job',
  refresh='load',
  multi=true,
  includeAll=true,
  allValues='.+',
  sort=1,
);

local hostname_template = grafana.template.new(
  'hostname',
  '$prometheus_datasource',
  'label_values(windows_cs_hostname{job=~"$job"}, hostname)',
  label='Hostname',
  refresh='load',
  multi=true,
  allValues='.+',
  sort=1,
);

{
  grafanaDashboards+:: {
    'windows_exporter.json':
      grafana.dashboard.new(
        'Windows overview',
        time_from='%s' % $._config.dashboardPeriod,
        editable=false,
        tags=($._config.dashboardTags),
        timezone='%s' % $._config.dashboardTimezone,
        refresh='%s' % $._config.dashboardRefresh,
        uid='windows-overview',
      )

      .addTemplates([
        ds_template,
        job_template,
        hostname_template,
      ])

      .addLink(grafana.link.dashboards(
        asDropdown=false,
        title='Windows dashboards',
        includeVars=true,
        keepTime=true,
        tags=($._config.dashboardTags),
      ))


      // Status Row
      .addPanel(grafana.row.new(title='Integration status'), gridPos={ x: 0, y: 0, w: 0, h: 0 })
      // Integration status
      .addPanel(integration_status_panel, gridPos={ x: 0, y: 0, w: 8, h: 2 })
      // Latest metric received
      .addPanel(latest_metric_panel, gridPos={ x: 8, y: 0, w: 8, h: 2 })

      .addPanel(grafana.row.new(title='Overview'), gridPos={ x: 0, y: 2, w: 0, h: 0 })
      .addPanel(usageTable, gridPos={ x: 0, y: 2, w: 24, h: 8 })

      .addPanel(grafana.row.new(title='Overview graphs'), gridPos={ x: 0, y: 10, w: 0, h: 0 })
      .addPanel(perCpu, gridPos={ x: 0, y: 10, w: 12, h: 6 })
      .addPanel(perMemory, gridPos={ x: 12, y: 10, w: 12, h: 6 })

      .addPanel(grafana.row.new(title='Resource details'), gridPos={ x: 0, y: 16, w: 0, h: 0 })
      .addPanel(uptime, gridPos={ x: 0, y: 16, w: 8, h: 4 })
      .addPanel(errorService, gridPos={ x: 8, y: 16, w: 8, h: 4 })
      .addPanel(diskUsage, gridPos={ x: 16, y: 16, w: 8, h: 4 })
      .addPanel(diskIO, gridPos={ x: 0, y: 20, w: 8, h: 8 })
      .addPanel(networkUsage, gridPos={ x: 8, y: 20, w: 8, h: 8 })
      .addPanel(iisConnections, gridPos={ x: 16, y: 20, w: 8, h: 8 }),

  },

  local integration_status_panel =
    grafana.statPanel.new(
      'Integration status',
      datasource='$prometheus_datasource',
      colorMode='background',
      graphMode='none',
      noValue='No Data',
      reducerFunction='lastNotNull'
    )
    .addMappings(
      [
        {
          options: {
            from: 1,
            result: {
              color: 'green',
              index: 0,
              text: 'Agent configured - sending metrics',
            },
            to: 10000000000000,
          },
          type: 'range',
        },
        {
          options: {
            from: 0,
            result: {
              color: 'red',
              index: 1,
              text: 'No Data',
            },
            to: 0,
          },
          type: 'range',
        },
      ]
    )
    .addTarget(
      grafana.prometheus.target('sum(windows_cpu_time_total{' + host_matcher + ',mode="idle"})')
    ),

  local latest_metric_panel =
    grafana.statPanel.new(
      'Latest metric received',
      datasource='$prometheus_datasource',
      colorMode='background',
      fields='Time',
      graphMode='none',
      noValue='No Data',
      reducerFunction='lastNotNull'
    )
    .addTarget(
      grafana.prometheus.target('sum(windows_cpu_time_total{' + host_matcher + ',mode="idle"})')
    ),

  local usageTable =
    win.wintable('Usage', '$prometheus_datasource')
    .addQuery('windows_os_info{' + host_matcher + '} * on(instance) group_right(product) windows_cs_hostname', 'group', 'group')
    .addQuery('100 - (avg by (instance) (rate(windows_cpu_time_total{' + host_matcher + ',mode="idle"}[$__rate_interval])) * 100)', 'CPU Usage %', 'cpuusage')
    .addQuery('time() - windows_system_system_up_time{' + host_matcher + '}', 'Uptime', 'uptime')
    .addQuery('windows_cs_logical_processors{' + host_matcher + '} - 0', 'CPUs', 'cpus')
    .addQuery('windows_cs_physical_memory_bytes{' + host_matcher + '} - 0', 'Memory', 'memory')
    .addQuery('100 - 100 * windows_os_physical_memory_free_bytes{' + host_matcher + '} / windows_cs_physical_memory_bytes{' + host_matcher + '}', 'Memory Used', 'memoryused')
    .addQuery('(windows_logical_disk_free_bytes{' + host_matcher + ',volume=~"C:"}/windows_logical_disk_size_bytes{' + host_matcher + ',volume=~"C:"}) * 100', 'C:\\ Free %', 'cfree')
    .hideColumn('Time')
    .hideColumn('domain')
    .hideColumn('fqdn')
    .hideColumn('job')
    .hideColumn('agent_hostname')
    .hideColumn('Value #group')
    .hideColumn('instance')
    .renameColumn('Value #cpuusage', 'CPU usage %')
    .renameColumn('hostname', 'Hostname')
    .renameColumn('product', 'OS version')
    .addThreshold('CPU usage %', [
      {
        color: 'dark-green',
        value: 0,
      },
      {
        color: 'dark-yellow',
        value: 40,
      },
      {
        color: 'dark-red',
        value: 80,
      },
    ], 'absolute')
    .renameColumn('Value #uptime', 'Uptime')
    .setColumnUnit('Uptime', 's')
    .addThreshold('Uptime', [
      {
        color: 'dark-red',
        value: 0,
      },
      {
        color: 'dark-yellow',
        value: 259200,
      },
      {
        color: 'dark-green',
        value: 432000,
      },
    ], 'absolute')
    .renameColumn('Value #cpus', 'CPUs')
    .renameColumn('Value #memory', 'Total memory')
    .setColumnUnit('Total memory', 'bytes')
    .renameColumn('Value #memoryused', 'Memory used %')
    .addThreshold('Memory used %', [
      {
        color: 'dark-green',
        value: 0,
      },
      {
        color: 'dark-yellow',
        value: 60,
      },
      {
        color: 'dark-red',
        value: 80,
      },
    ], 'absolute')
    .renameColumn('Value #cfree', 'C:\\ free %')
    .hideColumn('volume')
    .addThreshold('C:\\ free %', [
      {
        color: 'dark-red',
        value: null,
      },
      {
        color: 'dark-yellow',
        value: 20,
      },
      {
        color: 'dark-green',
        value: 80,
      },
    ], 'absolute'),

  local perCpu =
    graphPanel.new(
      title='CPU usage % by host',
      datasource='$prometheus_datasource',
      span=6,
      min=0,
      max=1,
      legend_show=false,
      percentage=true,
      format='percentunit'
    )
    .addTarget(prometheus.target(
      expr='1 - (avg by (agent_hostname) ( rate(windows_cpu_time_total{' + host_matcher + ',mode="idle"}[$__rate_interval])) )',
      legendFormat='{{agent_hostname}}',
      intervalFactor=2,
    )),

  local perMemory =
    graphPanel.new(
      title='Memory usage % by host',
      datasource='$prometheus_datasource',
      span=6,
      min=0,
      max=1,
      legend_show=false,
      percentage=true,
      format='percentunit'
    )
    .addTarget(prometheus.target(
      expr='1 - windows_os_physical_memory_free_bytes{' + host_matcher + '} / windows_cs_physical_memory_bytes{' + host_matcher + '}',
      legendFormat='{{agent_hostname}}',
    )),


  local iisConnections =
    graphPanel.new(
      title='IIS connections',
      datasource='$prometheus_datasource',
      span=3,
    )
    .addTarget(prometheus.target(
      expr='windows_iis_current_connections{' + host_matcher + '}',
      legendFormat='{{agent_hostname}}',
    )),

  local diskUsage =
    win.winbargauge('Usage of each partition', [
      {
        color: 'green',
        value: null,
      },
      {
        color: '#EAB839',
        value: 80,
      },
      {
        color: 'red',
        value: 90,
      },
    ], '100 - (windows_logical_disk_free_bytes{' + host_matcher + '} / windows_logical_disk_size_bytes{' + host_matcher + '})*100', '{{volume}}'),

  local diskIO =
    graphPanel.new(
      title='Disk read write',
      datasource='$prometheus_datasource',
      legend_min=true,
      legend_max=true,
      legend_avg=true,
      legend_current=true,
      legend_alignAsTable=true,
      legend_values=true,
      format='Bps',
      span=2
    )
    .addTarget(prometheus.target(
      expr='rate(windows_logical_disk_write_bytes_total{' + host_matcher + '}[$__rate_interval])',
      legendFormat='write {{volume}}',
    ))
    .addTarget(prometheus.target(
      expr='rate(windows_logical_disk_read_bytes_total{' + host_matcher + '}[$__rate_interval])',
      legendFormat='read {{volume}}',
    )),

  local errorService =
    win.winstat(
      span=1,
      title='Services in error',
      datasource='$prometheus_datasource',
      unit='short',
      overrides=[
        {
          matcher: {
            id: 'byFrameRefID',
            options: 'A',
          },
          properties: [
            {
              id: 'thresholds',
              value: {
                mode: 'absolute',
                steps: [

                  {
                    color: 'dark-green',
                    value: 0,
                  },
                  {
                    color: 'dark-red',
                    value: 1,
                  },
                ],
              },
            },
            {
              id: 'color',
            },
          ],
        },
      ],
    )
    .addTarget(prometheus.target(
      expr='sum(windows_service_status{status="error",' + host_matcher + '})',
      instant=true,

    )),

  local networkUsage =
    graphPanel.new(
      title='Network usage',
      datasource='$prometheus_datasource',
      legend_min=true,
      legend_max=true,
      legend_avg=true,
      legend_current=true,
      legend_alignAsTable=true,
      legend_values=true,
      format='percent',
      bars=true,
      lines=false,
      min=0,
      max=100,
      span=3
    )
    .addTarget(prometheus.target(
      expr='(rate(windows_net_bytes_total{' + host_matcher + ',nic!~"isatap.*|VPN.*"}[$__rate_interval]) * 8 / windows_net_current_bandwidth{' + host_matcher + ',nic!~"isatap.*|VPN.*"}) * 100',
      legendFormat='{{nic}}',
    )),


  local uptime =
    win.winstat(
      span=1,
      title='Uptime',
      datasource='$prometheus_datasource',
      format='s',
      overrides=[
        {
          matcher: {
            id: 'byFrameRefID',
            options: 'A',
          },
          properties: [
            {
              id: 'thresholds',
              value: {
                mode: 'absolute',
                steps: [
                  {
                    color: 'dark-red',
                    value: null,
                  },
                  {
                    color: 'dark-yellow',
                    value: 259200,
                  },
                  {
                    color: 'dark-green',
                    value: 432000,
                  },
                ],
              },
            },
            {
              id: 'color',
            },
          ],
        },
      ],
    )
    .addTarget(prometheus.target(
      expr='time() - windows_system_system_up_time{' + host_matcher + '}',
      instant=true,

    )),

}
