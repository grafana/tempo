local utils = import '../lib/utils.libsonnet';
local g = (import 'grafana-builder/grafana.libsonnet');
local grafana = (import 'grafonnet/grafana.libsonnet');

local host_matcher = 'job_snmp=~"$job", instance=~"$instance"';
local snmp_target_matcher = host_matcher + ', snmp_target=~"$snmp_target"';
local interface_matcher = snmp_target_matcher + ', ifDescr=~"$interface"';

local queries = {
  up_time: 'sysUpTime{' + snmp_target_matcher + '} * 10',
  max_out_current: 'max(irate(ifHCOutOctets{' + interface_matcher + '}[$__rate_interval]))',
  max_in_current: 'max(irate(ifHCInOctets{' + interface_matcher + '}[$__rate_interval]))',
  total_out: 'max(delta(ifHCOutOctets{' + interface_matcher + '}[$__range]))',
  total_in: 'max(delta(ifHCInOctets{' + interface_matcher + '}[$__range]))',
  oper_status: 'ifOperStatus{' + interface_matcher + '}',
  interface_out: 'irate(ifHCOutOctets{' + interface_matcher + '}[$__rate_interval])',
  interface_in: '-irate(ifHCInOctets{' + interface_matcher + '}[$__rate_interval])',
  interface_out_errors: 'ifOutErrors{' + interface_matcher + '}',
  interface_in_errors: 'ifInErrors{' + interface_matcher + '}',
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
  'label_values(snmp_scrape_duration_seconds, job_snmp)',
  label='Job',
  refresh='load',
  multi=false,
  includeAll=false,
  sort=1,
);

local instance_template = grafana.template.new(
  'instance',
  '$prometheus_datasource',
  'label_values(snmp_scrape_duration_seconds{job_snmp=~"$job"}, instance)',
  label='Instance',
  refresh='load',
  multi=false,
  includeAll=false,
  sort=1,
);

local snmp_target_template = grafana.template.new(
  'snmp_target',
  '$prometheus_datasource',
  'label_values(snmp_scrape_duration_seconds{job_snmp=~"$job", instance=~"$instance"}, snmp_target)',
  label='SNMP Target',
  refresh='load',
  multi=false,
  includeAll=false,
  sort=1,
);

local interface_template = grafana.template.new(
  'interface',
  '$prometheus_datasource',
  'label_values(ifType_info{job_snmp=~"$job", instance=~"$instance", snmp_target=~"$snmp_target"}, ifDescr)',
  label='Interface',
  refresh='load',
  multi=true,
  includeAll=true,
  sort=1,
);

// Panels
local up_time_panel =
  grafana.statPanel.new(
    'System Uptime',
    description='The time since the network management portion of the system was last re-initialized.',
    unit='ms',
    graphMode='none',
    datasource='$prometheus_datasource',
    reducerFunction='lastNotNull',
  )
  .addTarget(
    grafana.prometheus.target(queries.up_time)
  );

local max_out_current_panel =
  grafana.statPanel.new(
    'Max Out (Current)',
    description='The maximum number of bytes transmitted out of all interfaces, including framing characters.',
    unit='decbytes',
    datasource='$prometheus_datasource',
    reducerFunction='lastNotNull',
  )
  .addTarget(
    grafana.prometheus.target(queries.max_out_current)
  );

local max_in_current_panel =
  grafana.statPanel.new(
    'Max In (Current)',
    description='The maximum number of bytes transmitted into all interfaces, including framing characters.',
    unit='decbytes',
    datasource='$prometheus_datasource',
    reducerFunction='lastNotNull',
  )
  .addTarget(
    grafana.prometheus.target(queries.max_in_current)
  );

local total_out_panel =
  grafana.statPanel.new(
    'Total Out',
    description='The total number of bytes transmitted out of all interfaces, including framing characters.',
    unit='decbytes',
    graphMode='none',
    datasource='$prometheus_datasource',
    reducerFunction='lastNotNull',
  )
  .addTarget(
    grafana.prometheus.target(queries.total_out)
  );

local total_in_panel =
  grafana.statPanel.new(
    'Total In',
    description='The total number of bytes transmitted into all interfaces, including framing characters.',
    unit='decbytes',
    graphMode='none',
    datasource='$prometheus_datasource',
    reducerFunction='lastNotNull',
  )
  .addTarget(
    grafana.prometheus.target(queries.total_in)
  );

local oper_status_panel =
  grafana.statPanel.new(
    'Interface Operational Status',
    description='Shows the current operational state of the interface.',
    datasource='$prometheus_datasource',
    unit='string',
    colorMode='value',
    graphMode='none',
    noValue='Unknown',
    reducerFunction='lastNotNull',
    justifyMode='auto',
    orientation='horizontal',
    textMode='value_and_name',
  )
  .addMappings(
    [
      {
        options: {
          '1': {
            color: 'green',
            index: 1,
            text: 'Up',
          },
        },
        type: 'value',
      },
      {
        options: {
          '2': {
            color: 'red',
            index: 1,
            text: 'Down',
          },
        },
        type: 'value',
      },
      {
        options: {
          '3': {
            color: 'blue',
            index: 1,
            text: 'Test',
          },
        },
        type: 'value',
      },
      {
        options: {
          '4': {
            color: 'white',
            index: 1,
            text: 'Unknown',
          },
        },
        type: 'value',
      },
    ]
  )
  .addTarget(
    grafana.prometheus.target(queries.oper_status, legendFormat='{{ifDescr}}')
  );

local interface_traffic_panel =
  grafana.graphPanel.new(
    'Per Interface Traffic (Current)',
    description='Current traffic per interface (In values are represented as negative).',
    datasource='$prometheus_datasource',
    span=6,
  )
  .addTarget(
    grafana.prometheus.target(queries.interface_out, legendFormat='{{ifDescr}} Out')
  )
  .addTarget(
    grafana.prometheus.target(queries.interface_in, legendFormat='{{ifDescr}} In')
  ) +
  utils.timeSeriesOverride(
    unit='decbytes',
    fillOpacity=10,
    lineInterpolation='smooth',
    showPoints='never',
  );

local interface_out_errors_panel =
  grafana.statPanel.new(
    'Total Errors Out',
    description='The number of outbound packets that contained errors preventing them from being deliverable to a higher-layer protocol.',
    unit='short',
    graphMode='none',
    datasource='$prometheus_datasource',
    reducerFunction='lastNotNull',
  )
  .addTarget(
    grafana.prometheus.target(queries.interface_out_errors, legendFormat='{{ifDescr}}')
  );

local interface_in_errors_panel =
  grafana.statPanel.new(
    'Total Errors In',
    description='the number of inbound packets that contained errors preventing them from being deliverable to a higher-layer protocol.',
    unit='short',
    graphMode='none',
    datasource='$prometheus_datasource',
    reducerFunction='lastNotNull',
  )
  .addTarget(
    grafana.prometheus.target(queries.interface_in_errors, legendFormat='{{ifDescr}}')
  );


local interface_info_panel =
  grafana.tablePanel.new(
    'Device Interfaces Information',
    description='General information of all the interfaces of the target device.',
    datasource='$prometheus_datasource',
    span=12,
    styles=[
      { alias: 'Interface', pattern: 'ifDescr' },
      { alias: 'Type', pattern: 'ifType' },
      { alias: 'Speed', pattern: 'Value #B', type: 'number', unit: 'bps' },
      { alias: 'MAC Address', pattern: 'ifPhysAddress' },
      { alias: 'MTU', pattern: 'Value #D', type: 'number', unit: 'none' },
    ],
  )
  .addTarget(grafana.prometheus.target(
    'count by (ifDescr, ifType) (ifType_info{' + interface_matcher + '})',
    format='table',
    instant=true,
  ))
  .addTarget(grafana.prometheus.target(
    'max by (ifDescr) (ifSpeed{' + interface_matcher + '})',
    format='table',
    instant=true,
  ))
  .addTarget(grafana.prometheus.target(
    'count by (ifDescr, ifPhysAddress) (ifPhysAddress{' + interface_matcher + '})',
    format='table',
    instant=true,
  ))
  .addTarget(grafana.prometheus.target(
    'max by (ifDescr) (ifMtu{' + interface_matcher + '})',
    format='table',
    instant=true,
  ))
  .hideColumn('Time')
  .hideColumn('Value #A')
  .hideColumn('Value #C');


// Manifested stuff starts here
{
  grafanaDashboards+:: {
    'snmp-overview.json':
      grafana.dashboard.new(
        'SNMP Overview',
        time_from='%s' % $._config.dashboardPeriod,
        editable=false,
        tags=($._config.dashboardTags),
        timezone='%s' % $._config.dashboardTimezone,
        refresh='%s' % $._config.dashboardRefresh,
        uid='integration-snmp-overview'
      )

      .addTemplates([
        ds_template,
        job_template,
        instance_template,
        snmp_target_template,
        interface_template,
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
      // System Uptime
      .addPanel(up_time_panel, gridPos={ x: 0, y: 2, w: 4, h: 4 })
      // Max Out Current
      .addPanel(max_out_current_panel, gridPos={ x: 4, y: 2, w: 4, h: 4 })
      // Max in Current
      .addPanel(max_in_current_panel, gridPos={ x: 8, y: 2, w: 4, h: 4 })
      // Total Out
      .addPanel(total_out_panel, gridPos={ x: 12, y: 2, w: 4, h: 4 })
      // Total In
      .addPanel(total_in_panel, gridPos={ x: 16, y: 2, w: 4, h: 4 })

      // Interfaces Overview
      .addPanel(grafana.row.new(title='Interfaces Overview'), gridPos={ x: 0, y: 6, w: 0, h: 0 })
      // Interface Operational Statuses
      .addPanel(oper_status_panel, gridPos={ x: 0, y: 6, w: 6, h: 8 })
      // Interface Traffic
      .addPanel(interface_traffic_panel, gridPos={ x: 6, y: 6, w: 18, h: 8 })

      // Interface info Row
      .addPanel(grafana.row.new(title='Interface Specific Information'), gridPos={ x: 0, y: 14, w: 0, h: 0 })
      // Device Interfaces Information
      .addPanel(interface_info_panel, gridPos={ x: 0, y: 14, w: 32, h: 6 })

      // Interfaces Errors
      .addPanel(grafana.row.new(title='Interfaces Errors'), gridPos={ x: 0, y: 20, w: 0, h: 0 })
      // Interface Operational Statuses
      .addPanel(interface_out_errors_panel, gridPos={ x: 0, y: 20, w: 12, h: 4 })
      // Interface Traffic
      .addPanel(interface_in_errors_panel, gridPos={ x: 12, y: 20, w: 12, h: 4 }),
  },
}
