{
  grafanaDashboards+:: {
    'nomad-jobs.json':

      {
        annotations: {
          list: [
            {
              builtIn: 1,
              datasource: '-- Grafana --',
              enable: true,
              hide: true,
              iconColor: 'rgba(0, 211, 255, 1)',
              name: 'Annotations & Alerts',
              target: {
                limit: 100,
                matchAny: false,
                tags: [],
                type: 'dashboard',
              },
              type: 'dashboard',
            },
          ],
        },
        description: 'Nomad jobs metrics',
        editable: true,
        fiscalYearStartMonth: 0,
        gnetId: 6281,
        graphTooltip: 0,
        id: 111,
        iteration: 1651176094947,
        links: [],
        liveNow: false,
        panels: [
          {
            collapsed: false,
            gridPos: {
              h: 1,
              w: 24,
              x: 0,
              y: 0,
            },
            id: 9,
            panels: [],
            repeat: 'instance',
            title: '$instance',
            type: 'row',
          },
          {
            datasource: {
              uid: '$datasource',
            },
            fieldConfig: {
              defaults: {
                color: {
                  mode: 'palette-classic',
                },
                custom: {
                  axisLabel: '',
                  axisPlacement: 'auto',
                  barAlignment: 0,
                  drawStyle: 'line',
                  fillOpacity: 10,
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
                  spanNulls: true,
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
                unit: 'percent',
              },
              overrides: [],
            },
            gridPos: {
              h: 6,
              w: 12,
              x: 0,
              y: 1,
            },
            id: 2,
            links: [],
            options: {
              legend: {
                calcs: [],
                displayMode: 'list',
                placement: 'bottom',
              },
              tooltip: {
                mode: 'multi',
                sort: 'desc',
              },
            },
            pluginVersion: '8.4.7',
            targets: [
              {
                expr: 'avg(nomad_client_allocs_cpu_total_percent{instance=~"$instance"}) by(exported_job, task)',
                format: 'time_series',
                interval: '',
                intervalFactor: 1,
                legendFormat: '{{task}}',
                refId: 'A',
              },
            ],
            title: 'CPU usage',
            type: 'timeseries',
          },
          {
            datasource: {
              type: 'prometheus',
              uid: '$datasource',
            },
            fieldConfig: {
              defaults: {
                color: {
                  mode: 'palette-classic',
                },
                custom: {
                  axisLabel: '',
                  axisPlacement: 'auto',
                  barAlignment: 0,
                  drawStyle: 'line',
                  fillOpacity: 10,
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
                  spanNulls: true,
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
                unit: 'timeticks',
              },
              overrides: [],
            },
            gridPos: {
              h: 6,
              w: 12,
              x: 12,
              y: 1,
            },
            id: 3,
            links: [],
            options: {
              legend: {
                calcs: [],
                displayMode: 'list',
                placement: 'bottom',
              },
              tooltip: {
                mode: 'multi',
                sort: 'desc',
              },
            },
            pluginVersion: '8.4.7',
            targets: [
              {
                datasource: {
                  type: 'prometheus',
                  uid: '1_UFfQJGk',
                },
                exemplar: true,
                expr: 'avg(nomad_client_allocs_cpu_total_ticks{instance=~"$instance"}) by (exported_job, task)',
                format: 'time_series',
                interval: '',
                intervalFactor: 1,
                legendFormat: '{{task}}',
                refId: 'A',
              },
            ],
            title: 'CPU total ticks',
            type: 'timeseries',
          },
          {
            datasource: {
              uid: '$datasource',
            },
            fieldConfig: {
              defaults: {
                color: {
                  mode: 'palette-classic',
                },
                custom: {
                  axisLabel: '',
                  axisPlacement: 'auto',
                  barAlignment: 0,
                  drawStyle: 'line',
                  fillOpacity: 10,
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
                  spanNulls: true,
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
                unit: 'decbytes',
              },
              overrides: [],
            },
            gridPos: {
              h: 6,
              w: 12,
              x: 0,
              y: 7,
            },
            id: 6,
            links: [],
            options: {
              legend: {
                calcs: [],
                displayMode: 'list',
                placement: 'bottom',
              },
              tooltip: {
                mode: 'multi',
                sort: 'desc',
              },
            },
            pluginVersion: '8.4.7',
            targets: [
              {
                expr: 'avg(nomad_client_allocs_memory_rss{instance=~"$instance"}) by(exported_job, task)',
                format: 'time_series',
                interval: '',
                intervalFactor: 1,
                legendFormat: '{{task}}',
                refId: 'A',
              },
            ],
            title: 'RSS',
            type: 'timeseries',
          },
          {
            id: 7,
            gridPos: {
              h: 6,
              w: 12,
              x: 12,
              y: 7,
            },
            type: 'timeseries',
            title: 'Memory cache',
            datasource: {
              type: 'prometheus',
              uid: '$datasource',
            },
            pluginVersion: '8.4.7',
            links: [],
            options: {
              tooltip: {
                mode: 'multi',
                sort: 'desc',
              },
              legend: {
                displayMode: 'list',
                placement: 'bottom',
                calcs: [],
              },
            },
            targets: [
              {
                datasource: {
                  type: 'prometheus',
                  uid: '1_UFfQJGk',
                },
                exemplar: true,
                expr: 'avg(nomad_client_allocs_memory_cache{instance=~"$instance"}) by (exported_job, task)',
                format: 'time_series',
                interval: '',
                intervalFactor: 1,
                legendFormat: '{{task}}',
                refId: 'A',
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
                unit: 'decbytes',
              },
              overrides: [],
            },
            timeFrom: null,
            timeShift: null,
          },
        ],
        schemaVersion: 35,
        style: 'dark',
        tags: $._config.dashboardTags,
        templating: {
          list: [
            {
              current: {
                selected: false,
                text: 'Prometheus',
                value: 'Prometheus',
              },
              hide: 0,
              includeAll: false,
              label: 'Data Source',
              multi: false,
              name: 'datasource',
              options: [],
              query: 'prometheus',
              refresh: 1,
              regex: '',
              skipUrlSync: false,
              type: 'datasource',
            },
            {
              current: {
                selected: false,
                text: 'dc1',
                value: 'dc1',
              },
              datasource: {
                uid: '$datasource',
              },
              definition: '',
              hide: 0,
              includeAll: false,
              label: 'DC',
              multi: false,
              name: 'datacenter',
              options: [],
              query: {
                query: 'label_values(nomad_client_uptime, datacenter)',
                refId: 'prometheus-datacenter-Variable-Query',
              },
              refresh: 1,
              regex: '',
              skipUrlSync: false,
              sort: 0,
              tagValuesQuery: '',
              tagsQuery: '',
              type: 'query',
              useTags: false,
            },
            {
              current: {
                selected: true,
                text: [
                  'All',
                ],
                value: [
                  '$__all',
                ],
              },
              datasource: {
                uid: '$datasource',
              },
              definition: '',
              hide: 0,
              includeAll: true,
              label: 'Nomad cilent',
              multi: true,
              name: 'instance',
              options: [],
              query: {
                query: 'label_values(nomad_client_uptime{datacenter=~"$datacenter"}, instance)',
                refId: 'prometheus-instance-Variable-Query',
              },
              refresh: 2,
              regex: '',
              skipUrlSync: false,
              sort: 0,
              tagValuesQuery: '',
              tagsQuery: '',
              type: 'query',
              useTags: false,
            },
          ],
        },
        time: {
          from: 'now-5m',
          to: 'now',
        },
        timepicker: {
          refresh_intervals: [
            '5s',
            '10s',
            '30s',
            '1m',
            '5m',
            '15m',
            '30m',
            '1h',
            '2h',
            '1d',
          ],
          time_options: [
            '5m',
            '15m',
            '1h',
            '6h',
            '12h',
            '24h',
            '2d',
            '7d',
            '30d',
          ],
        },
        timezone: '',
        title: 'Nomad jobs',
        uid: 'TvqbbhViz',
        version: 56,
        weekStart: '',
      },
  },
}
