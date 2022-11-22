local grafana = (import 'grafonnet/grafana.libsonnet');
local dashboard = grafana.dashboard;
local template = grafana.template;
local dashboardUid = 'discourse-overview';

local prometheus = grafana.prometheus;
local promDatasourceName = 'prometheus_datasource';

local promDatasource = {
  uid: '${%s}' % promDatasourceName,
};

local overviewRow = {
  collapsed: false,
  title: 'Overview',
  type: 'row',
};

local trafficPanel = {
  datasource: promDatasource,
  description: 'Rate of HTTP traffic over time for the entire application. Grouped by response code.',
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
      'sum(rate(discourse_http_requests{instance=~"$instance",job=~"$job"}[$__rate_interval])) by (api,status,verb)',
      datasource=promDatasource,
    ),
  ],
  title: 'Traffic by Response Code',
  type: 'timeseries',
};

local activeRequests = {
  datasource: promDatasource,
  description: 'Active web requests for the entire application',
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
      'discourse_active_app_reqs{instance=~"$instance",job=~"$job"}',
      datasource=promDatasource,
    ),
  ],
  title: 'Active Requests',
  type: 'timeseries',
};

local queuedRequestsPanel = {
  datasource: promDatasource,
  description: 'Queued web requests for the entire application.',
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
      'discourse_queued_app_reqs{instance=~"$instance",job=~"$job"}',
      datasource=promDatasource,
    ),
  ],
  title: 'Queued Requests',
  type: 'timeseries',
};

local pageviewsPanel = {
  datasource: promDatasource,
  description: 'Rate of pageviews for the entire application. Grouped by type and service.',
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
      mappings: [
        {
          options: {
            match: 'null',
            result: {
              text: 'N/A',
            },
          },
          type: 'special',
        },
      ],
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
      unit: 'views/sec',
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
      'rate(discourse_page_views{instance=~"$instance",job=~"$job"}[$__rate_interval])',
      datasource=promDatasource,
    ),
  ],
  title: 'Page Views',
  type: 'timeseries',
};
local latencyRow = {
  collapsed: false,
  title: 'Latency',
  type: 'row',
};
local medianLatencyPanel = {
  datasource: promDatasource,
  description: 'The median amount of time for “latest” page requests for the selected site.',
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
      unit: 's',
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
      'sum(discourse_http_duration_seconds{quantile="0.5",action="latest",instance=~"$instance",job=~"$job"}) by (controller)',
      datasource=promDatasource,
    ),
  ],
  title: 'Latest Median Request Time',
  type: 'timeseries',
};

local topicMedianPanel = {
  datasource: promDatasource,
  description: 'The median amount of time for “topics show” requests for the selected site.',
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
      unit: 's',
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
      'sum(discourse_http_duration_seconds{quantile="0.5",controller="topics",instance=~"$instance",job=~"$job"}) by (controller)',
      datasource=promDatasource
    ),
  ],
  title: 'Topic Show Median Request Time',
  type: 'timeseries',
};
local ninetyNinthPercentileRequestLatency = {
  datasource: promDatasource,
  description: 'The 99th percentile amount of time for “latest” page requests for the selected site.',
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
      unit: 's',
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
    {
      datasource: promDatasource,
      editorMode: 'code',
      expr: 'sum(discourse_http_duration_seconds{quantile="0.99",action="latest",instance=~"$instance",job=~"$job"}) by (controller)',
      legendFormat: '__auto',
      range: true,
      refId: 'A',
    },
  ],
  title: 'Latest 99th percentile Request Time',
  type: 'timeseries',
};
local ninetyNinthTopicShowPercentileRequestLatency = {
  datasource: promDatasource,
  description: 'The 99th percentile amount of time for “topics show” requests for the selected site.',
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
      unit: 's',
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
      'discourse_http_duration_seconds{quantile="0.99",controller="topics",instance=~"$instance",job=~"$job"}',
      datasource=promDatasource,
    ),
  ],
  title: 'Topic Show 99th percentile Request Time',
  type: 'timeseries',
};

{
  grafanaDashboards+:: {
    'discourse-overview.json':
      dashboard.new(
        'Discourse Overview',
        time_from='%s' % $._config.dashboardPeriod,
        editable=false,
        tags=($._config.dashboardTags),
        timezone='%s' % $._config.dashboardTimezone,
        refresh='%s' % $._config.dashboardRefresh,
        graphTooltip='shared_crosshair',
        uid=dashboardUid,
      )
      .addLink(grafana.link.dashboards(
        asDropdown=false,
        title='Other discourse dashboards',
        includeVars=true,
        keepTime=true,
        tags=($._config.dashboardTags),
      )).addTemplates(
        [
          {
            hide: 0,
            label: 'Data source',
            name: 'prometheus_datasource',
            query: 'prometheus',
            refresh: 1,
            regex: '',
            type: 'datasource',
          },
          template.new(
            name='instance',
            label='instance',
            datasource='$prometheus_datasource',
            query='label_values(discourse_page_views{}, instance)',
            current='',
            refresh=2,
            includeAll=true,
            multi=false,
            allValues='.+',
            sort=1
          ),
          template.new(
            name='job',
            datasource=promDatasource,
            query='label_values(discourse_page_views{}, job)',
            label='Job',
            refresh='time',
            includeAll=true,
            multi=false,
            allValues='.+',
            sort=1
          ),
        ]
      )
      .addPanels(
        std.flattenArrays([
          [
            overviewRow { gridPos: { h: 1, w: 24, x: 0, y: 0 } },
            trafficPanel { gridPos: { h: 8, w: 24, x: 0, y: 1 } },
            activeRequests { gridPos: { h: 8, w: 12, x: 0, y: 9 } },
            queuedRequestsPanel { gridPos: { h: 8, w: 12, x: 12, y: 9 } },
            pageviewsPanel { gridPos: { h: 9, w: 24, x: 0, y: 18 } },
          ],
          // next row
          [
            latencyRow { gridPos: { h: 1, w: 24, x: 0, y: 25 } },
            medianLatencyPanel { gridPos: { h: 8, w: 12, x: 0, y: 26 } },
            topicMedianPanel { gridPos: { h: 8, w: 12, x: 12, y: 26 } },
            ninetyNinthPercentileRequestLatency { gridPos: { h: 8, w: 12, x: 0, y: 34 } },
            ninetyNinthTopicShowPercentileRequestLatency { gridPos: { h: 8, w: 12, x: 12, y: 34 } },
          ],
        ])
      ),
  },
}
