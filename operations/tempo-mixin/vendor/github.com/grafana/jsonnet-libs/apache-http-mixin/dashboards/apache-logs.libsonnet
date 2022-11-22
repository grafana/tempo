local grafana = import 'github.com/grafana/grafonnet-lib/grafonnet/grafana.libsonnet';
local dashboard = grafana.dashboard;
local template = grafana.template;
local dashboardUid = 'apache-http-logs';
local matcher = 'job=~"$job", instance=~"$instance"';

local errorLogRegex = |||
  `^\[[^ ]* (?P<timestamp>[^\]]*)\] \[(?:(?P<module>[^:\]]+):)?(?P<level>[^\]]+)\](?: \[pid (?P<pid>[^\]]*)\])?(?: \[client (?P<client>[^\]]*)\])? (?P<message>.*)$`
|||;
local accessLogregex = |||
  `^(?P<ip>[^ ]*) [^ ]* (?P<user>[^ ]*) \[(?P<timestamp>[^\]]*)\] "(?P<method>\S+)(?: +(?P<path>[^ ]*) +\S*)?" (?P<code>[^ ]*) (?P<size>[^ ]*)(?: "(?P<referer>[^\"]*)" "(?P<agent>.*)")?$`
|||;
local logsByLevel =
  {
    type: 'timeseries',  // barchart vs timeseries
    title: 'Logs by level',
    datasource: {
      uid: '${loki_datasource}',
      type: 'loki',
    },
    fieldConfig: {
      defaults: {
        unit: 'short',
        custom: {
          drawStyle: 'bars',
        },
        color: {
          mode: 'palette-classic',
        },
      },
      overrides: [
        {
          matcher: {
            id: 'byName',
            options: 'error',
          },
          properties: [
            {
              id: 'color',
              value: {
                mode: 'fixed',
                fixedColor: 'red',
              },
            },
          ],
        },
        {
          matcher: {
            id: 'byName',
            options: 'notice',
          },
          properties: [
            {
              id: 'color',
              value: {
                mode: 'fixed',
                fixedColor: 'green',
              },
            },
          ],
        },
      ],
    },
    targets: [
      {
        expr: 'sum(count_over_time({' + matcher + ', logtype="error", level!=""}[$__interval])) by (level)',
        legendFormat: '{{ level }}',
      },
    ],
    options: {
      tooltip: {
        mode: 'multi',
        sort: 'none',
      },
      legend: {
        displayMode: 'list',
        placement: 'bottom',
        calcs: ['sum'],
      },
    },
  };

local logsByHTTPcodes =
  {
    type: 'timeseries',
    title: 'Logs by HTTP codes',
    datasource: {
      uid: '${prometheus_datasource}',
      type: 'prometheus',
    },
    fieldConfig: {
      defaults: {
        unit: 'short',
        custom: {
          drawStyle: 'bars',
          lineInterpolation: 'linear',
          lineWidth: 1,
          fillOpacity: 60,
          gradientMode: 'none',
          spanNulls: false,
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
      },
      overrides: [
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
            options: 'HTTP 200-299',
          },
          properties: [
            {
              id: 'color',
              value: {
                fixedColor: 'green',
                mode: 'fixed',
              },
            },
          ],
        },
        {
          matcher: {
            id: 'byName',
            options: 'HTTP 400-499',
          },
          properties: [
            {
              id: 'color',
              value: {
                mode: 'fixed',
                fixedColor: 'orange',
              },
            },
          ],
        },
        {
          matcher: {
            id: 'byName',
            options: 'HTTP 100-199',
          },
          properties: [
            {
              id: 'color',
              value: {
                mode: 'fixed',
                fixedColor: 'purple',
              },
            },
          ],
        },
      ],
    },
    targets: [
      {
        expr: |||
          label_replace(
            sum by (le,job, instance) (increase(apache_response_http_codes_bucket{le!="+Inf", %s}[$__rate_interval])),
            "alias", "HTTP ${1}00-${1}99", "le", "(.).+"
          )
        ||| % matcher,
        legendFormat: '{{ alias }}',
        format: 'heatmap',
      },
    ],
    options: {
      tooltip: {
        mode: 'multi',
        sort: 'none',
      },
      legend: {
        displayMode: 'list',
        placement: 'bottom',
        calcs: ['sum'],
      },
    },
  };
{
  grafanaDashboards+::

    if $._config.enableLokiLogs then {
      'apache-http-logs.json':
        dashboard.new(
          'Apache HTTP server logs',
          time_from='%s' % $._config.dashboardPeriod,
          editable=false,
          tags=($._config.dashboardTags),
          timezone='%s' % $._config.dashboardTimezone,
          refresh='%s' % $._config.dashboardRefresh,
          uid=dashboardUid,
        )
        .addLink(grafana.link.dashboards(
          asDropdown=false,
          title='Other Apache HTTP dashboards',
          includeVars=true,
          keepTime=true,
          tags=($._config.dashboardTags),
        ))
        .addTemplates(
          [
            {
              hide: 0,
              label: 'Metrics datasource',
              name: 'prometheus_datasource',
              query: 'prometheus',
              refresh: 1,
              regex: '',
              type: 'datasource',
            },
            {
              hide: 0,
              label: 'Loki datasource',
              name: 'loki_datasource',
              query: 'loki',
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
          [
            grafana.row.new('Error logs') { gridPos: { y: 0 } },
            logsByLevel { gridPos: { w: 24, y: 0, h: 6 } },
            grafana.logPanel.new(
              title='Apache error logs',
              datasource='$loki_datasource',
            ).addTarget(grafana.loki.target('{' + matcher + ', logtype="error"}'))
            // \n| regexp ' + errorLogRegex))
            + { gridPos: { w: 24, y: 1, h: 8 } },

            grafana.row.new('Access logs') { gridPos: { y: 2 } },
            logsByHTTPcodes { gridPos: { w: 24, y: 2, h: 6 } },
            grafana.logPanel.new(
              title='Apache access logs',
              datasource='$loki_datasource',
            )
            .addTarget(grafana.loki.target('{' + matcher + ', logtype="access"}'))
            // \n| regexp ' + accessLogregex))
            + { gridPos: { w: 24, y: 2, h: 8 } },
          ]
        ),
    } else {},
}
