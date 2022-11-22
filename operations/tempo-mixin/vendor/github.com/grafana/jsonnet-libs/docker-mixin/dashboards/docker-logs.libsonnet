local g = (import 'grafana-builder/grafana.libsonnet');
local grafana = (import 'grafonnet/grafana.libsonnet');
local custom_barchart_grafonnet = import '../lib/custom-barchart-grafonnet/custom-barchart.libsonnet';

local host_matcher = 'job=~"$job", instance=~"$instance"';
local container_matcher = host_matcher + ', container=~"$container"';

local queries = {
  total_log_lines: 'sum(count_over_time({' + container_matcher + '}[$__interval]))',
  total_log_warnings: 'sum(count_over_time({' + container_matcher + '} |= "Warning" [$__interval]))',
  total_log_errors: 'sum(count_over_time({' + container_matcher + '} |= "Error" [$__interval]))',
  error_percentage: 'sum( count_over_time({' + container_matcher + '} |= "Error" [$__interval]) ) / sum( count_over_time({' + container_matcher + '} [$__interval]) )',
  total_bytes: 'sum(bytes_over_time({' + container_matcher + '} [$__interval]))',
  error_log_lines: '{' + container_matcher + '} |= "Error"',
  warning_log_lines: '{' + container_matcher + '} |= "Warning"',
  log_full_lines: '{' + container_matcher + '}',
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
  label: 'Prometheus Data Source',
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
  'label_values(machine_scrape_error, job)',
  label='Job',
  refresh='load',
  multi=true,
  includeAll=true,
  allValues='.+',
  sort=1,
  regex=''
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
  regex=''
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
local total_log_lines_panel =
  grafana.statPanel.new(
    'Total Log Lines',
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
    'Error Percentage',
    datasource='$loki_datasource',
    graphMode='none',
    reducerFunction='lastNotNull',
    unit='percent',
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
    'Bytes Used',
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
    'Full Log File',
    datasource='$loki_datasource',
  )
  .addTarget(
    grafana.loki.target(queries.log_full_lines)
  );

// Manifested stuff starts here
{
  grafanaDashboards+:: {
    'docker-logs.json':
      grafana.dashboard.new(
        'Docker Logs',
        time_from='%s' % $._config.dashboardPeriod,
        editable=false,
        tags=($._config.dashboardTags),
        timezone='%s' % $._config.dashboardTimezone,
        refresh='%s' % $._config.dashboardRefresh,
        uid='integration-docker-logs'
      )

      .addTemplates([
        prometheus_template,
        loki_template,
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
        grafana.row.new(title='Complete Log File', collapse=true)
        // Full Log File
        .addPanel(log_full_panel, gridPos={ x: 0, y: 28, w: 24, h: 8 }),
        gridPos={ x: 0, y: 28, w: 0, h: 0 }
      ),
  },
}
