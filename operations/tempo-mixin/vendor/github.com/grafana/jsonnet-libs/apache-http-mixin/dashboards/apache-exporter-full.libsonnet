local grafana = import 'github.com/grafana/grafonnet-lib/grafonnet/grafana.libsonnet';
local dashboard = grafana.dashboard;
local template = grafana.template;
local dashboardUid = 'apache-http';
local matcher = 'job=~"$job", instance=~"$instance"';

local uptimePanel =
  {
    datasource: {
      uid: '${prometheus_datasource}',
    },
    fieldConfig: {
      defaults: {
        color: {
          mode: 'thresholds',
        },
        decimals: 1,
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
        unit: 's',
      },
    },
    id: 8,
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
    pluginVersion: '8.4.5',
    targets: [
      {
        expr: 'apache_uptime_seconds_total{' + matcher + '}',
        format: 'time_series',
        intervalFactor: 1,
        step: 240,
      },
    ],
    title: 'Uptime',
    type: 'stat',
  };

local versionPanel =
  {
    type: 'stat',
    title: 'Version',
    datasource: {
      uid: '${prometheus_datasource}',
      type: 'prometheus',
    },
    pluginVersion: '8.4.5',
    maxDataPoints: 100,
    fieldConfig: {
      defaults: {
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
        color: {
          mode: 'thresholds',
        },
        decimals: 1,
        unit: 'none',
      },
    },
    options: {
      reduceOptions: {
        values: false,
        calcs: [
          'lastNotNull',
        ],
        fields: '',
      },
      orientation: 'horizontal',
      textMode: 'name',
      colorMode: 'none',
      graphMode: 'none',
      justifyMode: 'auto',
      text: {
        titleSize: 2,
      },
    },
    targets: [
      {
        expr: 'apache_info{' + matcher + '}',
        legendFormat: '{{ version }}',
        interval: '',
        exemplar: false,
        format: 'time_series',
        intervalFactor: 1,
        step: 240,
        instant: true,
      },
    ],
  };

local statusPanel =
  {

    type: 'state-timeline',
    title: 'Apache Up / Down',
    datasource: {
      uid: '${prometheus_datasource}',
      type: 'prometheus',
    },
    pluginVersion: '8.4.5',
    options: {
      mergeValues: false,
      showValue: 'never',
      alignValue: 'left',
      rowHeight: 0.9,
      legend: {
        displayMode: 'list',
        placement: 'right',
      },
      tooltip: {
        mode: 'single',
        sort: 'none',
      },
    },
    targets: [
      {
        expr: 'apache_up{' + matcher + '}',
        legendFormat: 'Apache up',
        interval: '',
        exemplar: true,
        format: 'time_series',
        intervalFactor: 1,
        refId: 'A',
        step: 240,
      },
    ],
    fieldConfig: {
      defaults: {
        custom: {
          lineWidth: 0,
          fillOpacity: 70,
          spanNulls: false,
        },
        color: {
          mode: 'continuous-GrYlRd',
        },
        mappings: [
          {
            type: 'value',
            options: {
              '0': {
                text: 'Down',
                color: 'red',
                index: 1,
              },
              '1': {
                text: 'Up',
                color: 'green',
                index: 0,
              },
            },
          },
        ],
        thresholds: {
          mode: 'absolute',
          steps: [
            {
              color: 'green',
              value: null,
            },
          ],
        },
      },
    },
  };
local responseTimePanel =
  {
    type: 'timeseries',
    title: 'Response time',
    datasource: {
      uid: '${prometheus_datasource}',
    },
    pluginVersion: '8.4.5',
    fieldConfig: {
      defaults: {
        custom: {
          drawStyle: 'line',
          lineInterpolation: 'linear',
          barAlignment: 0,
          lineWidth: 1,
          fillOpacity: 10,
          gradientMode: 'none',
          spanNulls: true,
          showPoints: 'never',
          pointSize: 5,
          stacking: {
            mode: 'none',
            group: 'A',
          },
          axisPlacement: 'auto',
          axisLabel: '',
          scaleDistribution: {
            type: 'linear',
          },
          hideFrom: {
            tooltip: false,
            viz: false,
            legend: false,
          },
          thresholdsStyle: {
            mode: 'off',
          },
        },
        color: {
          mode: 'palette-classic',
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
        unit: 'ms',
      },
    },
    options: {
      tooltip: {
        mode: 'multi',
        sort: 'none',
      },
      legend: {
        displayMode: 'table',
        placement: 'bottom',
        calcs: [
          'mean',
          'lastNotNull',
          'max',
          'min',
        ],
      },
    },
    targets: [
      {
        expr: 'increase(apache_duration_ms_total{' + matcher + '}[$__rate_interval])/increase(apache_accesses_total{' + matcher + '}[$__rate_interval])',
        legendFormat: 'Average response time',
        interval: '',
        exemplar: false,
        format: 'time_series',
        intervalFactor: 1,
        refId: 'A',
        step: 240,
        datasource: {
          uid: '${prometheus_datasource}',
          type: 'prometheus',
        },
      },
    ],
  };
local loadPanel =
  {

    type: 'timeseries',
    title: 'Load',
    datasource: {
      uid: '${prometheus_datasource}',
    },
    pluginVersion: '8.4.5',
    fieldConfig: {
      defaults: {
        custom: {
          drawStyle: 'line',
          lineInterpolation: 'linear',
          barAlignment: 0,
          lineWidth: 1,
          fillOpacity: 10,
          gradientMode: 'none',
          spanNulls: true,
          showPoints: 'never',
          pointSize: 5,
          stacking: {
            mode: 'none',
            group: 'A',
          },
          axisPlacement: 'auto',
          axisLabel: '',
          scaleDistribution: {
            type: 'linear',
          },
          hideFrom: {
            tooltip: false,
            viz: false,
            legend: false,
          },
          thresholdsStyle: {
            mode: 'off',
          },
          lineStyle: {
            fill: 'solid',
          },
        },
        color: {
          mode: 'palette-classic',
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
      overrides: [
        {
          matcher: {
            id: 'byName',
            options: 'Bytes sent',
          },
          properties: [
            {
              id: 'custom.axisPlacement',
              value: 'right',
            },
            {
              id: 'custom.drawStyle',
              value: 'bars',
            },
            {
              id: 'unit',
              value: 'Bps',
            },
          ],
        },
      ],
    },
    options: {
      tooltip: {
        mode: 'multi',
        sort: 'none',
      },
      legend: {
        displayMode: 'table',
        placement: 'bottom',
        calcs: [
          'mean',
          'lastNotNull',
          'max',
          'min',
        ],
      },
    },
    targets: [
      {
        expr: 'rate(apache_accesses_total{' + matcher + '}[$__rate_interval])',
        legendFormat: 'Calls',
        interval: '',
        exemplar: false,
        format: 'time_series',
        intervalFactor: 1,
        refId: 'A',
        step: 240,
        datasource: {
          type: 'prometheus',
          uid: '${prometheus_datasource}',
        },
      },
      {
        expr: 'rate(apache_sent_kilobytes_total{' + matcher + '}[$__rate_interval]) * 1000',
        legendFormat: 'Bytes sent',
        interval: '',
        exemplar: false,
        datasource: {
          uid: '${prometheus_datasource}',
          type: 'prometheus',
        },
        refId: 'B',
        hide: false,
      },
    ],
    description: '',
  };
local apacheScoreboardPanel =
  {

    type: 'timeseries',
    title: 'Apache scoreboard statuses',
    datasource: {
      uid: '${prometheus_datasource}',
    },
    pluginVersion: '8.4.5',
    links: [],
    options: {
      tooltip: {
        mode: 'multi',
        sort: 'desc',
      },
      legend: {
        displayMode: 'table',
        placement: 'right',
        calcs: [
          'mean',
          'lastNotNull',
          'max',
          'min',
        ],
        sortBy: 'Last *',
        sortDesc: true,
      },
    },
    targets: [
      {
        expr: 'apache_scoreboard{' + matcher + '}',
        format: 'time_series',
        intervalFactor: 1,
        legendFormat: '{{ state }}',
        refId: 'A',
        step: 240,
      },
    ],
    fieldConfig: {
      defaults: {
        custom: {
          drawStyle: 'line',
          lineInterpolation: 'stepAfter',
          barAlignment: 0,
          lineWidth: 1,
          fillOpacity: 10,
          gradientMode: 'none',
          spanNulls: true,
          showPoints: 'never',
          pointSize: 5,
          stacking: {
            mode: 'normal',
            group: 'A',
          },
          axisPlacement: 'auto',
          axisLabel: '',
          scaleDistribution: {
            type: 'linear',
          },
          hideFrom: {
            tooltip: false,
            viz: false,
            legend: false,
          },
          thresholdsStyle: {
            mode: 'off',
          },
        },
        color: {
          mode: 'palette-classic',
        },
        mappings: [],
        thresholds: {
          mode: 'absolute',
          steps: [
            {
              value: null,
              color: 'green',
            },
            {
              value: 80,
              color: 'red',
            },
          ],
        },
        unit: 'short',
      },
      overrides: [],
    },
    timeFrom: null,
    timeShift: null,
  };

local apacheWorkerStatusPanel =
  {
    type: 'timeseries',
    title: 'Apache worker statuses',
    datasource: {
      uid: '${prometheus_datasource}',
    },
    pluginVersion: '8.4.5',
    links: [],
    options: {
      tooltip: {
        mode: 'multi',
        sort: 'none',
      },
      legend: {
        displayMode: 'table',
        placement: 'bottom',
        calcs: [
          'mean',
          'lastNotNull',
          'max',
          'min',
        ],
      },
    },
    targets: [
      {
        expr: 'apache_workers{' + matcher + '}\n',
        format: 'time_series',
        intervalFactor: 1,
        legendFormat: '{{ state }}',
        step: 240,
      },
    ],
    fieldConfig: {
      defaults: {
        custom: {
          drawStyle: 'line',
          lineInterpolation: 'stepAfter',
          barAlignment: 0,
          lineWidth: 1,
          fillOpacity: 10,
          gradientMode: 'none',
          spanNulls: true,
          showPoints: 'never',
          pointSize: 5,
          stacking: {
            mode: 'normal',
            group: 'A',
          },
          axisPlacement: 'auto',
          axisLabel: '',
          scaleDistribution: {
            type: 'linear',
          },
          hideFrom: {
            tooltip: false,
            viz: false,
            legend: false,
          },
          thresholdsStyle: {
            mode: 'off',
          },
        },
        color: {
          mode: 'palette-classic',
        },
        mappings: [],
        thresholds: {
          mode: 'absolute',
          steps: [
            {
              value: null,
              color: 'green',
            },
            {
              value: 80,
              color: 'red',
            },
          ],
        },
        unit: 'short',
      },
    },
  };
local apacheCpuPanel =
  {

    type: 'timeseries',
    title: 'Apache CPU load',
    datasource: {
      uid: '${prometheus_datasource}',
    },
    pluginVersion: '8.4.5',
    links: [],
    options: {
      tooltip: {
        mode: 'multi',
        sort: 'none',
      },
      legend: {
        displayMode: 'table',
        placement: 'bottom',
        calcs: [
          'mean',
          'lastNotNull',
          'max',
          'min',
        ],
      },
    },
    targets: [
      {
        expr: 'apache_cpuload{' + matcher + '}',
        format: 'time_series',
        intervalFactor: 1,
        legendFormat: 'Load',
        refId: 'A',
        step: 240,
      },
    ],
    fieldConfig: {
      defaults: {
        custom: {
          drawStyle: 'line',
          lineInterpolation: 'linear',
          barAlignment: 0,
          lineWidth: 1,
          fillOpacity: 10,
          gradientMode: 'none',
          spanNulls: true,
          showPoints: 'never',
          pointSize: 5,
          stacking: {
            mode: 'none',
            group: 'A',
          },
          axisPlacement: 'auto',
          axisLabel: '',
          scaleDistribution: {
            type: 'linear',
          },
          hideFrom: {
            tooltip: false,
            viz: false,
            legend: false,
          },
          thresholdsStyle: {
            mode: 'off',
          },
        },
        color: {
          mode: 'palette-classic',
        },
        mappings: [],
        thresholds: {
          mode: 'absolute',
          steps: [
            {
              value: null,
              color: 'green',
            },
            {
              value: 80,
              color: 'red',
            },
          ],
        },
        unit: 'short',
        min: 0,
      },
      overrides: [],
    },
  };
local errorsPanel =
  {
    type: 'timeseries',
    title: 'Errors rate',
    datasource: {
      uid: '${prometheus_datasource}',
      type: 'prometheus',
    },
    pluginVersion: '8.4.5',
    fieldConfig: {
      defaults: {
        custom: {
          drawStyle: 'bars',
          lineInterpolation: 'linear',
          barAlignment: 0,
          lineWidth: 0,
          fillOpacity: 57,
          gradientMode: 'opacity',
          spanNulls: true,
          showPoints: 'never',
          pointSize: 5,
          stacking: {
            mode: 'none',
            group: 'A',
          },
          axisPlacement: 'auto',
          axisLabel: '',
          scaleDistribution: {
            type: 'linear',
          },
          hideFrom: {
            tooltip: false,
            viz: false,
            legend: false,
          },
          thresholdsStyle: {
            mode: 'off',
          },
        },
        color: {
          mode: 'palette-classic',
        },
        mappings: [],
        thresholds: {
          mode: 'absolute',
          steps: [
            {
              color: 'green',
              value: null,
            },
          ],
        },
        unit: 'reqps',
        max: 100,
        min: 0,
      },
      overrides: [
        {
          matcher: {
            id: 'byName',
            options: 'HTTP 400-499',
          },
          properties: [
            {
              id: 'color',
              value: {
                fixedColor: 'orange',
                mode: 'fixed',
              },
            },
          ],
        },
        {
          matcher: {
            id: 'byName',
            options: 'HTTP 500-599',
          },
          properties: [
            {
              id: 'color',
              value: {
                fixedColor: 'red',
                mode: 'fixed',
              },
            },
          ],
        },
        {
          matcher: {
            id: 'byName',
            options: 'Error rate',
          },
          properties: [
            {
              id: 'custom.drawStyle',
              value: 'line',
            },
            {
              id: 'unit',
              value: 'percent',
            },
            {
              id: 'color',
              value: {
                mode: 'fixed',
                fixedColor: 'red',
              },
            },
            {
              id: 'custom.fillOpacity',
              value: 50,
            },
            {
              id: 'custom.lineWidth',
              value: 1,
            },
            {
              id: 'custom.lineInterpolation',
              value: 'smooth',
            },
          ],
        },
      ],
    },
    options: {
      tooltip: {
        mode: 'multi',
        sort: 'none',
      },
      legend: {
        displayMode: 'table',
        placement: 'bottom',
        calcs: [
          'mean',
          'lastNotNull',
          'max',
          'min',
        ],
      },
    },
    targets: [
      {
        expr: 'label_replace(\n  sum by (le,job, instance) (rate(apache_response_http_codes_bucket{le=~"499|599", ' + matcher + '}[$__rate_interval])),\n  "alias", "HTTP ${1}00-${1}99", "le", "(.).+"\n)\n',
        legendFormat: '{{ alias }}',
        interval: '',
        exemplar: false,
        datasource: {
          type: 'prometheus',
          uid: '${prometheus_datasource}',
        },
        format: 'heatmap',
        intervalFactor: 1,
        refId: 'A',
        step: 240,
        hide: true,
      },
      {
        expr: 'avg by (le,job, instance)\n(\n(\n  increase(apache_response_http_codes_bucket{le=~"499", job=~"$job", instance=~"$instance"}[$__rate_interval])\n- ignoring(le)\n  increase(apache_response_http_codes_bucket{le=~"399", job=~"$job", instance=~"$instance"}[$__rate_interval])\n)\n/\nincrease(apache_response_http_codes_count{job=~"$job", instance=~"$instance"}[$__rate_interval]) * 100\n)',
        legendFormat: 'Error rate',
        interval: '',
        exemplar: true,
        datasource: {
          type: 'prometheus',
          uid: '${prometheus_datasource}',
        },
        format: 'time_series',
        intervalFactor: 1,
        refId: 'B',
        step: 240,
        hide: false,
        instant: false,
      },
    ],
    description: 'Ratio of 4xx and 5xx HTTP responses to all calls.',
  };
{
  grafanaDashboards+:: {

    'apache-http.json':
      dashboard.new(
        'Apache HTTP server',
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
        title='Other Apache HTTP dashboards',
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
            name='job',
            label='job',
            datasource='$prometheus_datasource',
            query='label_values(apache_up, job)',
            current='',
            refresh=2,
            includeAll=true,
            multi=true,
            allValues='.+',
            sort=1
          ),
          template.new(
            name='instance',
            label='instance',
            datasource='$prometheus_datasource',
            query='label_values(apache_up{job=~"$job"}, instance)',
            current='',
            refresh=2,
            includeAll=false,
            sort=1
          ),
        ]
      )
      .addPanels(
        std.flattenArrays([
          [
            uptimePanel { gridPos: { y: 0, x: 0, h: 3, w: 4 } },
            versionPanel { gridPos: { y: 0, h: 3, w: 4, x: 4 } },
            statusPanel { gridPos: { y: 0, h: 3, w: 16, x: 8 } },
          ],
          //next row
          if $._config.enableLokiLogs then [
            loadPanel { gridPos: { y: 1, h: 7, w: 8, x: 0 } },
            responseTimePanel { gridPos: { y: 1, h: 7, w: 8, x: 8 } },
            errorsPanel { gridPos: { y: 1, h: 7, w: 8, x: 16 } },

          ] else [
            loadPanel { gridPos: { y: 1, h: 7, w: 12, x: 0 } },
            responseTimePanel { gridPos: { y: 1, h: 7, w: 12, x: 12 } },
          ],
          [  //next row
            apacheScoreboardPanel { gridPos: { y: 2, h: 10, w: 24, x: 0 } },
            //next row
            apacheWorkerStatusPanel { gridPos: { y: 3, h: 10, w: 12, x: 0 } },
            apacheCpuPanel { gridPos: { y: 3, h: 10, w: 12, x: 12 } },
          ],
        ])
      ),

  },
}
