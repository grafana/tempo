local g = (import 'grafana-builder/grafana.libsonnet');
local grafana = (import 'grafonnet/grafana.libsonnet');
local custom_barchart_grafonnet = import '../lib/custom-barchart-grafonnet/custom-barchart.libsonnet';

local host_matcher = 'job=~"$job", agent_hostname=~"$hostname"';
local log_channel_matcher = host_matcher + ', channel=~"$channel", source=~"$source"';
local windows_event_parser = '| json | line_format "ProcessID: {{.execution_processId}} Source: {{.source}} EventID: {{.event_id}} Level: {{.levelText}}  Message: {{.message}}"';

local queries = {
  total_log_lines: 'sum(count_over_time({' + log_channel_matcher + '}[$__interval]))',
  total_log_warnings: 'sum(count_over_time({' + log_channel_matcher + '} |= "Warning" [$__interval]))',
  total_log_errors: 'sum(count_over_time({' + log_channel_matcher + '} |= "Error" [$__interval]))',
  error_percentage: 'sum(count_over_time({' + log_channel_matcher + '} |= "Error" [$__interval])) / sum(count_over_time({' + log_channel_matcher + '} [$__interval]))',
  total_bytes: 'sum(bytes_over_time({' + log_channel_matcher + '} [$__interval]))',
  error_log_lines: '{' + log_channel_matcher + '} |= "Error" ' + windows_event_parser,
  warning_log_lines: '{' + log_channel_matcher + '} |= "Warning" ' + windows_event_parser,
  log_full_lines: '{' + log_channel_matcher + '} ' + windows_event_parser,
};

local stackstyle = {
  line: 1,
  fill: 5,
  fillGradient: 10,
};

// Templates
local prometheus_template = {
  current: {
    text: 'default',
    value: 'default',
  },
  hide: 0,
  label: 'Prometheus data source',
  name: 'prometheus_datasource',
  options: [],
  query: 'prometheus',
  refresh: 1,
  regex: '',
  type: 'datasource',
};

local loki_template = {
  current: {
    text: 'default',
    value: 'default',
  },
  hide: 0,
  label: 'Loki Data Source',
  name: 'loki_datasource',
  options: [],
  query: 'loki',
  refresh: 1,
  regex: '',
  type: 'datasource',
};

local job_template = grafana.template.new(
  'job',
  '$prometheus_datasource',
  'label_values(windows_cpu_time_total, job)',
  label='Job',
  refresh='load',
  multi=true,
  includeAll=true,
  allValues='.+',
  sort=1,
  regex=''
);

local host_template = grafana.template.new(
  'hostname',
  '$prometheus_datasource',
  'label_values(windows_cpu_time_total{job=~"$job"}, agent_hostname)',
  label='Hostname',
  refresh='load',
  multi=true,
  includeAll=true,
  allValues='.+',
  sort=1,
  regex=''
);

local channel_template = grafana.template.new(
  'channel',
  '$loki_datasource',
  'label_values({job=~"$job", agent_hostname=~"$hostname"}, channel)',
  label='Channel',
  refresh='load',
  multi=true,
  includeAll=true,
  allValues='.+',
  sort=1,
);

local source_template = grafana.template.new(
  'source',
  '$loki_datasource',
  'label_values({job=~"$job", agent_hostname=~"$hostname", channel=~"$channel"}, source)',
  label='Source',
  refresh='load',
  multi=true,
  includeAll=true,
  allValues='.+',
  sort=1,
);

// Panels
local integration_status_panel =
  grafana.statPanel.new(
    'Integration status',
    datasource='$loki_datasource',
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
            text: 'Agent configured - sending logs',
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
    grafana.loki.target(queries.total_log_lines)
  );

local latest_metric_panel =
  grafana.statPanel.new(
    'Latest metric received',
    datasource='$loki_datasource',
    colorMode='background',
    fields='Time',
    graphMode='none',
    noValue='No Data',
    reducerFunction='lastNotNull'
  )
  .addTarget(
    grafana.loki.target(queries.total_log_lines)
  );

local total_log_lines_panel =
  grafana.statPanel.new(
    'Total log lines',
    datasource='$loki_datasource',
    graphMode='none',
    reducerFunction='sum',
    unit='short',
  )
  .addThreshold(
    { color: 'rgb(192, 216, 255)', value: 0 }
  )
  .addTarget(
    grafana.loki.target(queries.total_log_lines)
  );

local total_log_warnings_panel =
  grafana.statPanel.new(
    'Warnings',
    datasource='$loki_datasource',
    graphMode='none',
    reducerFunction='sum',
    unit='short',
  ).addThreshold(
    { color: 'rgb(255, 152, 48)', value: 0 }
  )
  .addTarget(
    grafana.loki.target(queries.total_log_warnings)
  );

local total_log_errors_panel =
  grafana.statPanel.new(
    'Errors',
    datasource='$loki_datasource',
    graphMode='none',
    reducerFunction='sum',
    unit='short',
  ).addThreshold(
    { color: 'rgb(242, 73, 92)', value: 0 }
  )
  .addTarget(
    grafana.loki.target(queries.total_log_errors)
  );

local error_percentage_panel =
  grafana.statPanel.new(
    'Error percentage',
    datasource='$loki_datasource',
    graphMode='none',
    reducerFunction='lastNotNull',
    unit='percentunit',
  ).addThresholds([
    { color: 'rgb(255, 166, 176)', value: 0 },
    { color: 'rgb(255, 115, 131)', value: 25 },
    { color: 'rgb(196, 22, 42)', value: 50 },
  ])
  .addTarget(
    grafana.loki.target(queries.error_percentage)
  );

local total_bytes_panel =
  grafana.statPanel.new(
    'Bytes used',
    datasource='$loki_datasource',
    graphMode='none',
    reducerFunction='sum',
    unit='bytes',
  )
  .addThreshold(
    { color: 'rgb(184, 119, 217)', value: 0 }
  )
  .addTarget(
    grafana.loki.target(queries.total_bytes)
  );

local historical_logs_errors_warnings_panel =
  custom_barchart_grafonnet.new(
    q1=queries.total_log_lines,
    q2=queries.total_log_warnings,
    q3=queries.total_log_errors,
  );

local log_errors_panel =
  grafana.logPanel.new(
    'Errors',
    datasource='$loki_datasource',
  )
  .addTarget(
    grafana.loki.target(queries.error_log_lines)
  );

local log_warnings_panel =
  grafana.logPanel.new(
    'Warnings',
    datasource='$loki_datasource',
  )
  .addTarget(
    grafana.loki.target(queries.warning_log_lines)
  );

local log_full_panel =
  grafana.logPanel.new(
    'Full log file',
    datasource='$loki_datasource',
  )
  .addTarget(
    grafana.loki.target(queries.log_full_lines)
  );

// Manifested stuff starts here
{
  grafanaDashboards+:: {
    'windows_logs.json':
      grafana.dashboard.new(
        'Windows logs',
        time_from='%s' % $._config.dashboardPeriod,
        editable=false,
        tags=($._config.dashboardTags),
        timezone='%s' % $._config.dashboardTimezone,
        refresh='%s' % $._config.dashboardRefresh,
        uid='windows-logs'
      )

      .addTemplates([
        prometheus_template,
        loki_template,
        job_template,
        host_template,
        channel_template,
        source_template,
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

      // Overview Row
      .addPanel(grafana.row.new(title='Overview'), gridPos={ x: 0, y: 2, w: 0, h: 0 })
      // Total Log Lines
      .addPanel(total_log_lines_panel, gridPos={ x: 0, y: 2, w: 4, h: 4 })
      // Warnings
      .addPanel(total_log_warnings_panel, gridPos={ x: 4, y: 2, w: 4, h: 4 })
      // Errors
      .addPanel(total_log_errors_panel, gridPos={ x: 8, y: 2, w: 4, h: 4 })
      // Error Percentage
      .addPanel(error_percentage_panel, gridPos={ x: 12, y: 2, w: 4, h: 4 })
      // Bytes Used
      .addPanel(total_bytes_panel, gridPos={ x: 16, y: 2, w: 4, h: 4 })
      // Historical Logs / Warnings / Errors
      .addPanel(historical_logs_errors_warnings_panel, gridPos={ x: 0, y: 6, w: 24, h: 6 })

      // Errors Row
      .addPanel(
        grafana.row.new(title='Errors', collapse=true)
        // Errors
        .addPanel(log_errors_panel, gridPos={ x: 0, y: 12, w: 24, h: 8 }),
        gridPos={ x: 0, y: 12, w: 0, h: 0 }
      )


      // Warnings Row
      .addPanel(
        grafana.row.new(title='Warnings', collapse=true)
        // Warnings
        .addPanel(log_warnings_panel, gridPos={ x: 0, y: 20, w: 24, h: 8 }),
        gridPos={ x: 0, y: 20, w: 0, h: 0 }
      )

      // Complete Log File
      .addPanel(
        grafana.row.new(title='Complete log file', collapse=true)
        // Full Log File
        .addPanel(log_full_panel, gridPos={ x: 0, y: 28, w: 24, h: 8 }),
        gridPos={ x: 0, y: 28, w: 0, h: 0 }
      ),
  },
}
