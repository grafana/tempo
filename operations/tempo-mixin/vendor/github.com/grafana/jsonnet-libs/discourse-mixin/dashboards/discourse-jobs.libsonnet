local grafana = (import 'grafonnet/grafana.libsonnet');
local dashboard = grafana.dashboard;
local template = grafana.template;
local dashboardUid = 'discourse-jobs';

local prometheus = grafana.prometheus;
local promDatasourceName = 'prometheus_datasource';

local promDatasource = {
  uid: '${%s}' % promDatasourceName,
};

local skJobDurationPanel = {
  datasource: promDatasource,
  description: 'Time spent in Sidekiq jobs broken out by job name.',
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
        fillOpacity: 30,
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
        showPoints: 'never',
        spanNulls: false,
        stacking: {
          group: 'A',
          mode: 'normal',
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
  links: [],
  options: {
    legend: {
      calcs: [],
      displayMode: 'list',
      placement: 'bottom',
      showLegend: true,
    },
    tooltip: {
      mode: 'multi',
      sort: 'desc',
    },
  },
  pluginVersion: '9.1.8',
  targets: [
    prometheus.target(
      'sum(rate(discourse_sidekiq_job_duration_seconds{instance=~"$instance",job=~"$job"}[$__rate_interval])) by (job_name)',
      datasource=promDatasource,
      legendFormat='{{job_name}}'
    ),
  ],
  title: 'Sidekiq Job Duration',
  type: 'timeseries',
};

local sheduledJobDurationPanel = {
  datasource: promDatasource,
  description: 'Time spent in scheduled jobs broken out by job name.',
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
        fillOpacity: 30,
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
        showPoints: 'never',
        spanNulls: false,
        stacking: {
          group: 'A',
          mode: 'normal',
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
  links: [],
  options: {
    legend: {
      calcs: [],
      displayMode: 'list',
      placement: 'bottom',
      showLegend: true,
    },
    tooltip: {
      mode: 'multi',
      sort: 'desc',
    },
  },
  pluginVersion: '9.1.8',
  targets: [
    prometheus.target(
      'sum(rate(discourse_scheduled_job_duration_seconds{instance=~"$instance",job=~"$job"}[$__rate_interval])) by (job_name)',
      datasource=promDatasource,
      legendFormat='{{job_name}}',
    ),
  ],
  title: 'Scheduled Job Duration',
  type: 'timeseries',
};

local usedRSSMemoryPanel = {
  datasource: promDatasource,
  description: 'Total RSS Memory used by process. Broken up by pid.',
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
      unit: 'bytes',
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
      'sum(discourse_rss{instance=~"$instance",job=~"$job"}) by (pid)',
      datasource=promDatasource,
    ),
  ],
  title: 'Used RSS Memory',
  type: 'timeseries',
};

local v8HeapSizePanel = {
  datasource: promDatasource,
  description: 'Current heap size of V8 engine. Broken up by process type',
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
      unit: 'bytes',
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
      'sum(discourse_v8_used_heap_size{instance=~"$instance",job=~"$job"}) by (type)',
      datasource=promDatasource,
    ),
  ],
  title: 'V8 Heap Size',
  type: 'timeseries',
};

local skWorkerScore = {
  datasource: promDatasource,
  description: 'Current number of Sidekiq Workers.',
  fieldConfig: {
    defaults: {
      color: {
        mode: 'thresholds',
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
          },
          {
            color: 'red',
            value: 80,
          },
        ],
      },
      unit: 'none',
    },
    overrides: [],
  },
  links: [],
  maxDataPoints: 100,
  options: {
    colorMode: 'none',
    graphMode: 'none',
    justifyMode: 'auto',
    orientation: 'horizontal',
    reduceOptions: {
      calcs: [
        'lastNotNull',
      ],
      fields: '',
      values: false,
    },
    textMode: 'auto',
  },
  pluginVersion: '9.1.8',
  targets: [
    {
      datasource: promDatasource,
      editorMode: 'code',
      expr: 'count(discourse_rss{type="sidekiq",instance=~"$instance",job=~"$job"})',
      format: 'time_series',
      intervalFactor: 2,
      legendFormat: '',
      range: true,
      refId: 'A',
      step: 40,
      target: '',
    },
  ],
  title: 'Sidekiq Workers',
  type: 'stat',
};

local webWorkersStat = {
  datasource: promDatasource,
  description: 'Current number of Web Workers.',
  fieldConfig: {
    defaults: {
      color: {
        mode: 'thresholds',
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
          },
          {
            color: 'red',
            value: 80,
          },
        ],
      },
      unit: 'none',
    },
    overrides: [],
  },
  links: [],
  maxDataPoints: 100,
  options: {
    colorMode: 'none',
    graphMode: 'none',
    justifyMode: 'auto',
    orientation: 'horizontal',
    reduceOptions: {
      calcs: [
        'lastNotNull',
      ],
      fields: '',
      values: false,
    },
    textMode: 'auto',
  },
  pluginVersion: '9.1.8',
  targets: [
    {
      datasource: promDatasource,
      editorMode: 'code',
      expr: "count(discourse_rss{type='web',instance=~\"$instance\",job=~\"$job\"})",
      format: 'time_series',
      intervalFactor: 2,
      legendFormat: '',
      range: true,
      refId: 'A',
      step: 40,
      target: '',
    },
  ],
  title: 'Web Workers',
  type: 'stat',
};

local skQueuedStat = {
  datasource: promDatasource,
  description: 'Current number of jobs in Sidekiq queue.',
  fieldConfig: {
    defaults: {
      color: {
        mode: 'thresholds',
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
          },
          {
            color: 'red',
            value: 80,
          },
        ],
      },
      unit: 'none',
    },
    overrides: [],
  },
  links: [],
  maxDataPoints: 100,
  options: {
    colorMode: 'none',
    graphMode: 'none',
    justifyMode: 'auto',
    orientation: 'horizontal',
    reduceOptions: {
      calcs: [
        'lastNotNull',
      ],
      fields: '',
      values: false,
    },
    textMode: 'auto',
  },
  targets: [
    {
      datasource: promDatasource,
      editorMode: 'code',
      expr: 'max(discourse_sidekiq_jobs_enqueued{instance=~"$instance",job=~"$job"})',
      format: 'time_series',
      intervalFactor: 2,
      legendFormat: '',
      range: true,
      refId: 'A',
      step: 40,
      target: '',
    },
  ],
  title: 'Sidekiq Queued',
  type: 'stat',
};
{
  grafanaDashboards+:: {
    'discourse-jobs.json':
      dashboard.new(
        'Discourse Jobs Processing',
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
        title='Other Discourse dashboards',
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
            query='label_values(discourse_page_views, instance)',
            current='',
            refresh=2,
            includeAll=true,
            multi=false,
            allValues='.+',
            sort=1
          ),
          template.new(
            'job',
            promDatasource,
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
            skJobDurationPanel { gridPos: { h: 7, w: 12, x: 0, y: 0 } },
            sheduledJobDurationPanel { gridPos: { h: 7, w: 12, x: 12, y: 0 } },
          ],
          //next row
          [
            usedRSSMemoryPanel { gridPos: { h: 8, w: 12, x: 0, y: 7 } },
            v8HeapSizePanel { gridPos: { h: 8, w: 12, x: 12, y: 7 } },
          ],
          //next row
          [
            skWorkerScore { gridPos: { h: 6, w: 7, x: 0, y: 15 } },
            webWorkersStat { gridPos: { h: 6, w: 8, x: 7, y: 15 } },
            skQueuedStat { gridPos: { h: 6, w: 9, x: 15, y: 15 } },
          ],
        ])
      ),
  },
}
