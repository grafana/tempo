local g = (import 'grafana-builder/grafana.libsonnet');
local grafana = (import 'grafonnet/grafana.libsonnet');

local host_matcher = 'job=~"$job", instance=~"$instance"';
local container_matcher = host_matcher + ', name=~"$container"';

local queries = {
  total_containers: 'count(container_last_seen{' + container_matcher + '})',
  total_images: 'count (sum by (image) (container_last_seen{' + container_matcher + ', image=~".+"}))',
  host_mem_reserved: 'sum(container_spec_memory_reservation_limit_bytes{' + container_matcher + '}) / avg(machine_memory_bytes{' + host_matcher + '})',
  host_mem_consumed: 'sum(container_memory_usage_bytes{' + container_matcher + '}) / avg(machine_memory_bytes{' + host_matcher + '})',
  cpu_usage: 'sum (rate(container_cpu_usage_seconds_total{' + container_matcher + '}[$__rate_interval]))',
  cpu_by_container: 'avg by (name) (rate(container_cpu_usage_seconds_total{' + container_matcher + '}[$__rate_interval]))',
  mem_by_container: 'sum by (name) (container_memory_usage_bytes{' + container_matcher + '})',
  net_rx_by_container: 'sum by (name) (rate(container_network_receive_bytes_total{' + container_matcher + '}[$__rate_interval]))',
  net_tx_by_container: 'sum by (name) (rate(container_network_transmit_bytes_total{' + container_matcher + '}[$__rate_interval]))',
  net_rx_error_rate: 'sum(rate(container_network_receive_errors_total{' + container_matcher + '}[$__rate_interval]))',
  net_tx_error_rate: 'sum(rate(container_network_transmit_errors_total{' + container_matcher + '}[$__rate_interval]))',
  tcp_socket_by_state: 'sum(container_network_tcp_usage_total{' + container_matcher + '}) by (tcp_state) > 0',
  fs_usage_by_device: 'sum by (instance, device) (container_fs_usage_bytes{' + host_matcher + ', id="/", device=~"/dev/.+"} / container_fs_limit_bytes{' + host_matcher + ', id="/", device=~"/dev/.+"})',
  fs_inode_usage_by_device: '1 - sum by (instance, device) (container_fs_inodes_free{' + host_matcher + ', id="/", device=~"/dev/.+"} / container_fs_inodes_total{' + host_matcher + ', id="/", device=~"/dev/.+"})',
};

local stackstyle = {
  line: 1,
  fill: 5,
  fillGradient: 10,
};

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
  'label_values(machine_scrape_error, job)',
  label='Job',
  refresh='load',
  multi=true,
  includeAll=true,
  allValues='.+',
  sort=1,
);

local instance_template = grafana.template.new(
  'instance',
  '$prometheus_datasource',
  'label_values(machine_scrape_error{job=~"$job"}, instance)',
  label='Instance',
  refresh='load',
  multi=true,
  includeAll=true,
  allValues='.+',
  sort=1,
);

local container_template = grafana.template.new(
  'container',
  '$prometheus_datasource',
  'label_values(container_last_seen{job=~"$job", instance=~"$instance"}, name)',
  label='Container',
  refresh='load',
  multi=true,
  includeAll=true,
  allValues='.+',
  sort=1,
);

// Panels
local total_containers_panel =
  grafana.statPanel.new(
    'Total Containers',
    datasource='$prometheus_datasource',
    graphMode='none',
    reducerFunction='lastNotNull'
  )
  .addTarget(
    grafana.prometheus.target(queries.total_containers)
  );

local total_images_panel =
  grafana.statPanel.new(
    'Total Images',
    datasource='$prometheus_datasource',
    graphMode='none',
    reducerFunction='lastNotNull'
  )
  .addTarget(
    grafana.prometheus.target(queries.total_images)
  );

local cpu_usage_panel =
  grafana.singlestat.new(
    'CPU Utilization by Containers',
    format='percentunit',
    gaugeShow=true,
    thresholds='.80,.90',
    span=2,
    datasource='$prometheus_datasource',
    gaugeMaxValue=1,
  )
  .addTarget(
    grafana.prometheus.target(queries.cpu_usage)
  );

local mem_reserved_panel =
  grafana.singlestat.new(
    'Memory Reserved by Containers',
    format='percentunit',
    gaugeShow=true,
    thresholds='.80,.90',
    span=2,
    datasource='$prometheus_datasource',
    gaugeMaxValue=1,
  )
  .addTarget(
    grafana.prometheus.target(queries.host_mem_reserved)
  );

local mem_usage_panel =
  grafana.singlestat.new(
    'Memory Utilization by Containers',
    format='percentunit',
    gaugeShow=true,
    thresholds='.80,.90',
    span=2,
    datasource='$prometheus_datasource',
    gaugeMaxValue=1,
  )
  .addTarget(
    grafana.prometheus.target(queries.host_mem_consumed)
  );

local cpu_by_container_panel =
  grafana.graphPanel.new(
    'CPU',
    span=6,
    datasource='$prometheus_datasource',
  ) +
  g.queryPanel(
    [queries.cpu_by_container],
    ['{{name}}'],
  ) +
  g.stack +
  stackstyle +
  {
    yaxes: g.yaxes('percentunit'),
  };

local mem_by_container_panel =
  grafana.graphPanel.new(
    'Memory',
    span=6,
    datasource='$prometheus_datasource',
  ) +
  g.queryPanel(
    [queries.mem_by_container],
    ['{{name}}'],
  ) +
  g.stack +
  stackstyle +
  { yaxes: g.yaxes('bytes') };

local net_throughput_panel =
  grafana.graphPanel.new(
    'Bandwidth',
    span=6,
    datasource='$prometheus_datasource',
  ) +
  g.queryPanel(
    [queries.net_rx_by_container, queries.net_tx_by_container],
    ['{{name}} rx', '{{name}} tx'],
  ) +
  g.stack +
  stackstyle +
  {
    yaxes: g.yaxes({ format: 'binBps', min: null }),
  } + {
    seriesOverrides: [{ alias: '/.*tx/', transform: 'negative-Y' }],
  };

local tcp_socket_by_state_panel =
  grafana.graphPanel.new(
    'TCP Sockets By State',
    datasource='$prometheus_datasource',
    span=6,
  ) +
  g.queryPanel(
    [queries.tcp_socket_by_state],
    ['{{tcp_state}}'],
  ) +
  stackstyle;

local disk_usage_panel =
  g.tablePanel(
    [queries.fs_usage_by_device, queries.fs_inode_usage_by_device],
    {
      instance: { alias: 'Instance' },
      device: { alias: 'Device' },
      'Value #A': { alias: 'Disk Usage', unit: 'percentunit' },
      'Value #B': { alias: 'Inode Usage', unit: 'percentunit' },
    }
  ) + { span: 12, datasource: '$prometheus_datasource' };

// Manifested stuff starts here
{
  grafanaDashboards+:: {
    'docker.json':
      grafana.dashboard.new(
        'Docker Overview',
        time_from='%s' % $._config.dashboardPeriod,
        editable=false,
        tags=($._config.dashboardTags),
        timezone='%s' % $._config.dashboardTimezone,
        refresh='%s' % $._config.dashboardRefresh,
        uid='integration-docker-overview'
      )

      .addTemplates([
        ds_template,
        job_template,
        instance_template,
        container_template,
      ])

      .addLink(grafana.link.dashboards(
        asDropdown=false,
        title='Docker Dashboards',
        includeVars=true,
        keepTime=true,
        tags=($._config.dashboardTags),
      ))

      // Overview Row
      .addPanel(grafana.row.new(title='Overview'), gridPos={ x: 0, y: 2, w: 0, h: 0 })
      // Total containers
      .addPanel(total_containers_panel, gridPos={ x: 0, y: 2, w: 4, h: 6 })
      // Total Images
      .addPanel(total_images_panel, gridPos={ x: 4, y: 2, w: 4, h: 6 })
      // Host CPU used by containers
      .addPanel(cpu_usage_panel, gridPos={ x: 8, y: 2, w: 4, h: 6 })
      // Host memory reserved by containers
      .addPanel(mem_reserved_panel, gridPos={ x: 12, y: 2, w: 4, h: 6 })
      // Host memory utilization by containers
      .addPanel(mem_usage_panel, gridPos={ x: 16, y: 2, w: 4, h: 6 })

      // Compute Row
      .addPanel(grafana.row.new(title='Compute'), gridPos={ x: 0, y: 8, w: 0, h: 0 })
      // CPU by container
      .addPanel(cpu_by_container_panel, gridPos={ x: 0, y: 8, w: 12, h: 8 })
      // Memory by container
      .addPanel(mem_by_container_panel, gridPos={ x: 12, y: 8, w: 12, h: 8 })

      // Network Row
      .addPanel(grafana.row.new(title='Network'), gridPos={ x: 0, y: 16, w: 0, h: 0 })
      // Network throughput
      .addPanel(net_throughput_panel, gridPos={ x: 0, y: 16, w: 12, h: 8 })
      // TCP Socket by state
      .addPanel(tcp_socket_by_state_panel, gridPos={ x: 12, y: 16, w: 12, h: 8 })

      // Storage Row
      .addPanel(grafana.row.new(title='Storage'), gridPos={ x: 0, y: 24, w: 0, h: 0 })
      // Disk
      .addPanel(disk_usage_panel, gridPos={ x: 0, y: 24, w: 24, h: 8 }),
  },
}
