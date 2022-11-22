local g = (import 'grafana-builder/grafana.libsonnet');
local grafana = (import 'grafonnet/grafana.libsonnet');
local dashboard = grafana.dashboard;
local template = grafana.template;
local prometheus = grafana.prometheus;

local dashboardUid = 'gitlab-overview';
local matcher = 'job=~"$job", instance=~"$instance"';

local promDatasourceName = 'prometheus_datasource';
local lokiDatasourceName = 'loki_datasource';

local promDatasource = {
  uid: '${%s}' % promDatasourceName,
};

local lokiDatasource = {
  uid: '${%s}' % lokiDatasourceName,
};

local rowOverviewPanel = {
  collapsed: false,
  title: 'Overview',
  type: 'row',
};

local rowPipelineActivityPanel = {
  collapsed: false,
  title: 'Pipeline Activity',
  type: 'row',
};

local trafficByResponseCodePanel = {
  datasource: promDatasource,
  description: 'Rate of HTTP traffic over time, grouped by status.',
  fieldConfig: {
    defaults: {
      color: {
        mode: 'palette-classic',
      },
      custom: {
        axisCenteredZero: false,
        axisColorMode: 'text',
        axisLabel: '',
        axisPlacement: 'auto',
        barAlignment: 0,
        drawStyle: 'line',
        fillOpacity: 0,
        gradientMode: 'none',
        hideFrom: {
          legend: false,
          tooltip: false,
          viz: false,
        },
        lineInterpolation: 'linear',
        lineWidth: 1,
        pointSize: 5,
        scaleDistribution: {
          type: 'linear',
        },
        showPoints: 'auto',
        spanNulls: false,
        stacking: {
          group: 'A',
          mode: 'none',
        },
        thresholdsStyle: {
          mode: 'off',
        },
      },
      mappings: [],
      unit: 'reqps',
    },
    overrides: [],
  },
  options: {
    legend: {
      calcs: [],
      displayMode: 'list',
      placement: 'bottom',
      showLegend: true,
    },
    tooltip: {
      mode: 'single',
      sort: 'none',
    },
  },
  targets: [
    prometheus.target(
      'sum by (status) (rate(http_requests_total{%s}[$__rate_interval]))' % matcher,
      datasource=promDatasource,
      legendFormat='{{status}}'
    ),
  ],
  title: 'Traffic by Response Code',
  transformations: [],
  type: 'timeseries',
};

local userSessionsPanel = {
  datasource: promDatasource,
  description: 'The rate of user logins.',
  fieldConfig: {
    defaults: {
      color: {
        mode: 'palette-classic',
      },
      custom: {
        axisCenteredZero: false,
        axisColorMode: 'text',
        axisLabel: '',
        axisPlacement: 'auto',
        barAlignment: 0,
        drawStyle: 'line',
        fillOpacity: 0,
        gradientMode: 'none',
        hideFrom: {
          legend: false,
          tooltip: false,
          viz: false,
        },
        lineInterpolation: 'linear',
        lineWidth: 1,
        pointSize: 5,
        scaleDistribution: {
          type: 'linear',
        },
        showPoints: 'auto',
        spanNulls: false,
        stacking: {
          group: 'A',
          mode: 'none',
        },
        thresholdsStyle: {
          mode: 'off',
        },
      },
      mappings: [],
      unit: 'sessions/s',
    },
    overrides: [],
  },
  options: {
    legend: {
      calcs: [],
      displayMode: 'list',
      placement: 'bottom',
      showLegend: true,
    },
    tooltip: {
      mode: 'single',
      sort: 'none',
    },
  },
  targets: [
    prometheus.target(
      'rate(user_session_logins_total{%s}[$__rate_interval])' % matcher,
      datasource=promDatasource,
      legendFormat='sessions'
    ),
  ],
  title: 'User Sessions',
  type: 'timeseries',
};

local avgRequestLatencyPanel = {
  datasource: promDatasource,
  description: 'Average latency of inbound HTTP requests.',
  fieldConfig: {
    defaults: {
      color: {
        mode: 'palette-classic',
      },
      custom: {
        axisCenteredZero: false,
        axisColorMode: 'text',
        axisLabel: '',
        axisPlacement: 'auto',
        barAlignment: 0,
        drawStyle: 'line',
        fillOpacity: 0,
        gradientMode: 'none',
        hideFrom: {
          legend: false,
          tooltip: false,
          viz: false,
        },
        lineInterpolation: 'linear',
        lineWidth: 1,
        pointSize: 5,
        scaleDistribution: {
          type: 'linear',
        },
        showPoints: 'auto',
        spanNulls: false,
        stacking: {
          group: 'A',
          mode: 'none',
        },
        thresholdsStyle: {
          mode: 'off',
        },
      },
      mappings: [],
      unit: 'ms',
    },
    overrides: [],
  },
  options: {
    legend: {
      calcs: [],
      displayMode: 'list',
      placement: 'bottom',
      showLegend: true,
    },
    tooltip: {
      mode: 'single',
      sort: 'none',
    },
  },
  targets: [
    prometheus.target(
      '1000 * rate(http_request_duration_seconds_sum{%s}[$__rate_interval]) / rate(http_request_duration_seconds_count{%s}[$__rate_interval])' % [matcher, matcher],
      datasource=promDatasource,
      legendFormat='{{method}}'
    ),
  ],
  title: 'Average Request Latency',
  type: 'timeseries',
};

local top5ReqTypesPanel = {
  datasource: promDatasource,
  description: 'Top 5 types of requests to the server.',
  fieldConfig: {
    defaults: {
      color: {
        mode: 'thresholds',
      },
      custom: {
        align: 'left',
        displayMode: 'auto',
        filterable: false,
        inspect: false,
      },
      decimals: 3,
      mappings: [],
      thresholds: {
        mode: 'absolute',
        steps: [
          {
            color: 'green',
            value: null,
          },
          {
            color: 'red',
            value: 80,
          },
        ],
      },
      unit: 'reqps',
    },
    overrides: [
      {
        matcher: {
          id: 'byName',
          options: 'Method',
        },
        properties: [
          {
            id: 'custom.width',
            value: 270,
          },
        ],
      },
    ],
  },
  options: {
    footer: {
      fields: '',
      reducer: [
        'sum',
      ],
      show: false,
    },
    frameIndex: 1,
    showHeader: true,
    sortBy: [],
  },
  pluginVersion: '9.1.7',
  targets: [
    prometheus.target(
      'sum by (feature_category) (rate(http_requests_total{%s}[$__rate_interval]))' % matcher,
      format='table',
      datasource=promDatasource
    ),
  ],
  title: 'Top 5 Request Types',
  transformations: [
    {
      id: 'groupBy',
      options: {
        fields: {
          Value: {
            aggregations: [
              'lastNotNull',
            ],
            operation: 'aggregate',
          },
          feature_category: {
            aggregations: [
              'lastNotNull',
            ],
            operation: 'groupby',
          },
        },
      },
    },
    {
      id: 'organize',
      options: {
        excludeByName: {},
        indexByName: {},
        renameByName: {
          'Value (lastNotNull)': 'Request rate',
          feature_category: 'Feature category',
        },
      },
    },
    {
      id: 'sortBy',
      options: {
        fields: {},
        sort: [
          {
            desc: true,
            field: 'Request rate',
          },
        ],
      },
    },
    {
      id: 'limit',
      options: {
        limitField: 5,
      },
    },
  ],
  type: 'table',
};

local errorLogsPanel(dashboardRailsExceptionFilename) = {
  datasource: lokiDatasource,
  description: 'The GitLab rails error logs.',
  options: {
    dedupStrategy: 'none',
    enableLogDetails: true,
    prettifyLogMessage: false,
    showCommonLabels: false,
    showLabels: false,
    showTime: true,
    sortOrder: 'Descending',
    wrapLogMessage: false,
  },
  targets: [
    {
      datasource: lokiDatasource,
      editorMode: 'code',
      expr: '{filename="%s", %s} | json | line_format "{{.severity}} {{.exception_class}} - {{.exception_message}}"' % [dashboardRailsExceptionFilename, matcher],
      legendFormat: '',
      queryType: 'range',
      refId: 'A',
    },
  ],
  title: 'Error Logs',
  transformations: [],
  type: 'logs',
};

local jobActivationsPanel = {
  datasource: promDatasource,
  description: 'The number of jobs activated per second.',
  fieldConfig: {
    defaults: {
      color: {
        mode: 'palette-classic',
      },
      custom: {
        axisCenteredZero: false,
        axisColorMode: 'text',
        axisLabel: '',
        axisPlacement: 'auto',
        barAlignment: 0,
        drawStyle: 'line',
        fillOpacity: 0,
        gradientMode: 'none',
        hideFrom: {
          legend: false,
          tooltip: false,
          viz: false,
        },
        lineInterpolation: 'linear',
        lineWidth: 1,
        pointSize: 5,
        scaleDistribution: {
          type: 'linear',
        },
        showPoints: 'auto',
        spanNulls: false,
        stacking: {
          group: 'A',
          mode: 'none',
        },
        thresholdsStyle: {
          mode: 'off',
        },
      },
      mappings: [],
      unit: 'activations/s',
    },
    overrides: [],
  },
  options: {
    legend: {
      calcs: [],
      displayMode: 'list',
      placement: 'bottom',
      showLegend: true,
    },
    tooltip: {
      mode: 'single',
      sort: 'none',
    },
  },
  targets: [
    prometheus.target(
      'rate(gitlab_ci_active_jobs_sum{%s}[$__rate_interval])' % matcher,
      datasource=promDatasource,
      legendFormat='{{plan}}'
    ),
  ],
  title: 'Job Activations',
  type: 'timeseries',
};

local pipelinesCreatedPanel = {
  datasource: promDatasource,
  description: 'Rate of pipeline instances created.',
  fieldConfig: {
    defaults: {
      color: {
        mode: 'palette-classic',
      },
      custom: {
        axisCenteredZero: false,
        axisColorMode: 'text',
        axisLabel: '',
        axisPlacement: 'auto',
        barAlignment: 0,
        drawStyle: 'line',
        fillOpacity: 0,
        gradientMode: 'none',
        hideFrom: {
          legend: false,
          tooltip: false,
          viz: false,
        },
        lineInterpolation: 'linear',
        lineWidth: 1,
        pointSize: 5,
        scaleDistribution: {
          type: 'linear',
        },
        showPoints: 'auto',
        spanNulls: false,
        stacking: {
          group: 'A',
          mode: 'none',
        },
        thresholdsStyle: {
          mode: 'off',
        },
      },
      mappings: [],
      unit: 'pipelines/s',
    },
    overrides: [],
  },
  options: {
    legend: {
      calcs: [],
      displayMode: 'list',
      placement: 'bottom',
      showLegend: true,
    },
    tooltip: {
      mode: 'single',
      sort: 'none',
    },
  },
  targets: [
    prometheus.target(
      'rate(pipelines_created_total{%s}[$__rate_interval])' % matcher,
      datasource=promDatasource,
      legendFormat='{{source}}'
    ),
  ],
  title: 'Pipelines Created',
  type: 'timeseries',
};

local pipelineBuildsCreatedPanel = {
  datasource: promDatasource,
  description: 'The number of builds created within a pipeline per second, grouped by source.',
  fieldConfig: {
    defaults: {
      color: {
        mode: 'palette-classic',
      },
      custom: {
        axisCenteredZero: false,
        axisColorMode: 'text',
        axisLabel: '',
        axisPlacement: 'auto',
        barAlignment: 0,
        drawStyle: 'line',
        fillOpacity: 0,
        gradientMode: 'none',
        hideFrom: {
          legend: false,
          tooltip: false,
          viz: false,
        },
        lineInterpolation: 'linear',
        lineWidth: 1,
        pointSize: 5,
        scaleDistribution: {
          type: 'linear',
        },
        showPoints: 'auto',
        spanNulls: false,
        stacking: {
          group: 'A',
          mode: 'none',
        },
        thresholdsStyle: {
          mode: 'off',
        },
      },
      mappings: [],
      unit: 'builds/s',
    },
    overrides: [],
  },
  options: {
    legend: {
      calcs: [],
      displayMode: 'list',
      placement: 'bottom',
      showLegend: true,
    },
    tooltip: {
      mode: 'single',
      sort: 'none',
    },
  },
  targets: [
    prometheus.target(
      'sum by (source) (rate(gitlab_ci_pipeline_size_builds_sum{%s}[$__rate_interval]))' % matcher,
      datasource=promDatasource,
      legendFormat='{{source}}'
    ),
  ],
  title: 'Pipeline Builds Created',
  type: 'timeseries',
};

local buildTraceOperationsPanel = {
  datasource: promDatasource,
  description: 'The rate of build trace operations performed, grouped by source.',
  fieldConfig: {
    defaults: {
      color: {
        mode: 'palette-classic',
      },
      custom: {
        axisCenteredZero: false,
        axisColorMode: 'text',
        axisLabel: '',
        axisPlacement: 'auto',
        barAlignment: 0,
        drawStyle: 'line',
        fillOpacity: 0,
        gradientMode: 'none',
        hideFrom: {
          legend: false,
          tooltip: false,
          viz: false,
        },
        lineInterpolation: 'linear',
        lineWidth: 1,
        pointSize: 5,
        scaleDistribution: {
          type: 'linear',
        },
        showPoints: 'auto',
        spanNulls: false,
        stacking: {
          group: 'A',
          mode: 'none',
        },
        thresholdsStyle: {
          mode: 'off',
        },
      },
      mappings: [],
      unit: 'ops',
    },
    overrides: [],
  },
  options: {
    legend: {
      calcs: [],
      displayMode: 'list',
      placement: 'bottom',
      showLegend: true,
    },
    tooltip: {
      mode: 'single',
      sort: 'none',
    },
  },
  targets: [
    prometheus.target(
      'rate(gitlab_ci_trace_operations_total{%s}[$__rate_interval])' % matcher,
      datasource=promDatasource,
      legendFormat='{{operation}}'
    ),
  ],
  title: 'Build Trace Operations',
  type: 'timeseries',
};

{
  grafanaDashboards+:: {
    'gitlab-overview.json':
      dashboard.new(
        'GitLab Overview',
        time_from='%s' % $._config.dashboardPeriod,
        tags=($._config.dashboardTags),
        timezone='%s' % $._config.dashboardTimezone,
        refresh='%s' % $._config.dashboardRefresh,
        description='Overview of a GitLab instance.',
        uid=dashboardUid,
      )
      .addTemplates(std.flattenArrays([
        [
          template.datasource(
            promDatasourceName,
            'prometheus',
            null,
            label='Data Source',
            refresh='load'
          ),
          template.new(
            'job',
            promDatasource,
            'label_values(gitlab_rails_boot_time_seconds{}, job)',
            label='Job',
            refresh='time',
            includeAll=true,
            multi=true,
            allValues='.+',
            sort=1
          ),
          template.new(
            'instance',
            promDatasource,
            'label_values(gitlab_rails_boot_time_seconds{job=~"$job"}, instance)',
            label='Instance',
            refresh='time'
          ),
        ],
        if $._config.enableLokiLogs then [
          template.datasource(
            lokiDatasourceName,
            'loki',
            null,
            label='Loki Datasource',
            refresh='load'
          ),
        ] else [],
      ]))
      .addPanels(
        std.flattenArrays([
          [
            rowOverviewPanel { gridPos: { h: 1, w: 24, x: 0, y: 0 } },
            // Row 1
            trafficByResponseCodePanel { gridPos: { h: 9, w: 24, x: 0, y: 1 } },
            // Row 2
            userSessionsPanel { gridPos: { h: 9, w: 8, x: 0, y: 10 } },
            avgRequestLatencyPanel { gridPos: { h: 9, w: 8, x: 8, y: 10 } },
            top5ReqTypesPanel { gridPos: { h: 9, w: 8, x: 16, y: 10 } },
          ],
          if $._config.enableLokiLogs then [
            // Optional Row 3
            errorLogsPanel($._config.dashboardRailsExceptionFilename) { gridPos: { h: 8, w: 24, x: 0, y: 19 } },
          ] else [],
          [
            rowPipelineActivityPanel { gridPos: { h: 1, w: 24, x: 0, y: 27 } },
            // Row 1
            jobActivationsPanel { gridPos: { h: 9, w: 24, x: 0, y: 28 } },
            // Row 2
            pipelinesCreatedPanel { gridPos: { h: 8, w: 12, x: 0, y: 37 } },
            pipelineBuildsCreatedPanel { gridPos: { h: 8, w: 12, x: 12, y: 37 } },
            // Row 3
            buildTraceOperationsPanel { gridPos: { h: 8, w: 24, x: 0, y: 45 } },
          ],
        ])
      ),

  },
}
