local g = (import 'grafana-builder/grafana.libsonnet');
local grafana = (import 'grafonnet/grafana.libsonnet');
local dashboard = grafana.dashboard;
local template = grafana.template;
local dashboardUid = 'clickhouse-overview';
local matcher = 'job=~"$job", instance=~"$instance"';

local successfulQueriesPanel =
  {
    datasource: {
      type: 'prometheus',
      uid: '${prometheus_datasource}',
    },
    description: 'Rate of successful queries per second',
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
        unit: '/ sec',
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
        datasource: {
          type: 'prometheus',
          uid: '${prometheus_datasource}',
        },
        editorMode: 'builder',
        expr: 'rate(ClickHouseProfileEvents_Query{' + matcher + '}[$__rate_interval])',
        legendFormat: 'Query',
        range: true,
        refId: 'A',
      },
      {
        datasource: {
          type: 'prometheus',
          uid: '${prometheus_datasource}',
        },
        editorMode: 'builder',
        expr: 'rate(ClickHouseProfileEvents_SelectQuery{' + matcher + '}[$__rate_interval])',
        hide: false,
        legendFormat: 'Select Query',
        range: true,
        refId: 'B',
      },
      {
        datasource: {
          type: 'prometheus',
          uid: '${prometheus_datasource}',
        },
        editorMode: 'builder',
        expr: 'rate(ClickHouseProfileEvents_InsertQuery{' + matcher + '}[$__rate_interval])',
        hide: false,
        legendFormat: 'Insert Query',
        range: true,
        refId: 'C',
      },
      {
        datasource: {
          type: 'prometheus',
          uid: '${prometheus_datasource}',
        },
        editorMode: 'builder',
        expr: 'rate(ClickHouseProfileEvents_AsyncInsertQuery{' + matcher + '}[$__rate_interval])',
        hide: false,
        legendFormat: 'Async Insert Query',
        range: true,
        refId: 'D',
      },
    ],
    title: 'Successful Queries',
    type: 'timeseries',
  };

local failedQueriesPanel =
  {
    datasource: {
      type: 'prometheus',
      uid: '${prometheus_datasource}',
    },
    description: 'Rate of failed queries per second',
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
        unit: '/ sec',
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
        datasource: {
          type: 'prometheus',
          uid: '${prometheus_datasource}',
        },
        editorMode: 'builder',
        expr: 'rate(ClickHouseProfileEvents_FailedQuery{' + matcher + '}[$__rate_interval])',
        legendFormat: 'Failed Query',
        range: true,
        refId: 'A',
      },
      {
        datasource: {
          type: 'prometheus',
          uid: '${prometheus_datasource}',
        },
        editorMode: 'builder',
        expr: 'rate(ClickHouseProfileEvents_FailedSelectQuery{' + matcher + '}[$__rate_interval])',
        hide: false,
        legendFormat: 'Failed Select Query',
        range: true,
        refId: 'B',
      },
      {
        datasource: {
          type: 'prometheus',
          uid: '${prometheus_datasource}',
        },
        editorMode: 'builder',
        expr: 'rate(ClickHouseProfileEvents_FailedInsertQuery{' + matcher + '}[$__rate_interval])',
        hide: false,
        legendFormat: 'Failed Insert Query',
        range: true,
        refId: 'C',
      },
    ],
    title: 'Failed Queries',
    type: 'timeseries',
  };

local rejectedInsertsPanel =
  {
    datasource: {
      type: 'prometheus',
      uid: '${prometheus_datasource}',
    },
    description: 'Number of rejected inserts per second',
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
        unit: '/ sec',
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
        datasource: {
          type: 'prometheus',
          uid: '${prometheus_datasource}',
        },
        editorMode: 'builder',
        expr: 'rate(ClickHouseProfileEvents_RejectedInserts{' + matcher + '}[$__rate_interval])',
        legendFormat: 'Rejected Inserts',
        range: true,
        refId: 'A',
      },
    ],
    title: 'Rejected Inserts',
    type: 'timeseries',
  };

local memoryUsagePanel =
  {
    datasource: {
      type: 'prometheus',
      uid: '${prometheus_datasource}',
    },
    description: 'Memory usage over time',
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
        unit: 'decbytes',
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
        datasource: {
          type: 'prometheus',
          uid: '${prometheus_datasource}',
        },
        editorMode: 'builder',
        expr: 'ClickHouseMetrics_MemoryTracking{' + matcher + '}',
        legendFormat: 'Memory Tracking',
        range: true,
        refId: 'A',
      },
    ],
    title: 'Memory Usage',
    type: 'timeseries',
  };

local memoryUsageGaugePanel =
  {
    datasource: {
      type: 'prometheus',
      uid: '${prometheus_datasource}',
    },
    description: 'Percentage of memory allocated by Clickhouse compared to OS total',
    fieldConfig: {
      defaults: {
        color: {
          mode: 'thresholds',
        },
        mappings: [],
        max: 100,
        min: 0,
        thresholds: {
          mode: 'percentage',
          steps: [
            {
              color: 'green',
              value: null,
            },
            {
              color: '#EAB839',
              value: 80,
            },
            {
              color: 'red',
              value: 90,
            },
          ],
        },
        unit: 'percent',
      },
      overrides: [],
    },
    options: {
      orientation: 'auto',
      reduceOptions: {
        calcs: [
          'lastNotNull',
        ],
        fields: '',
        values: false,
      },
      showThresholdLabels: true,
      showThresholdMarkers: true,
      text: {},
    },
    pluginVersion: '9.1.6',
    targets: [
      {
        datasource: {
          type: 'prometheus',
          uid: '${prometheus_datasource}',
        },
        editorMode: 'code',
        expr: '(ClickHouseMetrics_MemoryTracking{' + matcher + '} / ClickHouseAsyncMetrics_OSMemoryTotal{' + matcher + '}) * 100',
        hide: false,
        legendFormat: '__auto',
        range: true,
        refId: 'A',
      },
    ],
    title: 'Memory Usage',
    type: 'gauge',
  };

local activeConnectionsPanel =
  {
    datasource: {
      type: 'prometheus',
      uid: '${prometheus_datasource}',
    },
    description: 'Current number of connections to Clickhouse',
    fieldConfig: {
      defaults: {
        color: {
          mode: 'palette-classic',
        },
        custom: {
          axisCenteredZero: false,
          axisColorMode: 'text',
          axisGridShow: true,
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
        unit: 'none',
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
        datasource: {
          type: 'prometheus',
          uid: '${prometheus_datasource}',
        },
        editorMode: 'builder',
        expr: 'ClickHouseMetrics_TCPConnection{' + matcher + '}',
        legendFormat: 'TCP Connection',
        range: true,
        refId: 'A',
      },
      {
        datasource: {
          type: 'prometheus',
          uid: '${prometheus_datasource}',
        },
        editorMode: 'builder',
        expr: 'ClickHouseMetrics_HTTPConnection{' + matcher + '}',
        hide: false,
        legendFormat: 'HTTP Connection',
        range: true,
        refId: 'B',
      },
      {
        datasource: {
          type: 'prometheus',
          uid: '${prometheus_datasource}',
        },
        editorMode: 'builder',
        expr: 'ClickHouseMetrics_MySQLConnection{' + matcher + '}',
        hide: false,
        legendFormat: 'MySQL Connection',
        range: true,
        refId: 'C',
      },
      {
        datasource: {
          type: 'prometheus',
          uid: '${prometheus_datasource}',
        },
        editorMode: 'builder',
        expr: 'ClickHouseMetrics_PostgreSQLConnection{' + matcher + '}',
        hide: false,
        legendFormat: 'PostgreSQL Connection',
        range: true,
        refId: 'D',
      },
    ],
    title: 'Active Connections',
    type: 'timeseries',
  };

local networkReceivedPanel =
  {
    datasource: {
      type: 'prometheus',
      uid: '${prometheus_datasource}',
    },
    description: 'Received network throughput',
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
        datasource: {
          type: 'prometheus',
          uid: '${prometheus_datasource}',
        },
        editorMode: 'builder',
        expr: 'rate(ClickHouseProfileEvents_NetworkReceiveBytes{' + matcher + '}[$__rate_interval])',
        legendFormat: 'Network Receive Bytes',
        range: true,
        refId: 'A',
      },
    ],
    title: 'Network Received',
    type: 'timeseries',
  };

local networkTransmittedPanel =
  {
    datasource: {
      type: 'prometheus',
      uid: '${prometheus_datasource}',
    },
    description: 'Transmitted network throughput',
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
        datasource: {
          type: 'prometheus',
          uid: '${prometheus_datasource}',
        },
        editorMode: 'builder',
        expr: 'rate(ClickHouseProfileEvents_NetworkSendBytes{' + matcher + '}[$__rate_interval])',
        legendFormat: 'Network Send Bytes',
        range: true,
        refId: 'A',
      },
    ],
    title: 'Network Transmitted',
    type: 'timeseries',
  };

local errorLogsPanel =
  {
    datasource: {
      type: 'loki',
      uid: '${loki_datasource}',
    },
    description: 'Recent logs from the error log file',
    options: {
      dedupStrategy: 'none',
      enableLogDetails: true,
      prettifyLogMessage: false,
      showCommonLabels: false,
      showLabels: false,
      showTime: false,
      sortOrder: 'Descending',
      wrapLogMessage: false,
    },
    targets: [
      {
        datasource: {
          type: 'loki',
          uid: '${loki_datasource}',
        },
        editorMode: 'builder',
        expr: '{filename="/var/log/clickhouse-server/clickhouse-server.err.log", %s}' % matcher,
        legendFormat: '',
        queryType: 'range',
        refId: 'A',
      },
    ],
    title: 'Error Logs',
    type: 'logs',
  };
{
  grafanaDashboards+:: {

    'clickhouse-overview.json':
      dashboard.new(
        'Clickhouse Overview',
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
        title='Other clickhouse dashboards',
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
            query='label_values(job)',
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
            query='label_values(ClickHouseProfileEvents_Query{job=~"$job"}, instance)',
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
            successfulQueriesPanel { gridPos: { h: 8, w: 24, x: 0, y: 0 } },
          ],
          //next row
          [
            failedQueriesPanel { gridPos: { h: 8, w: 12, x: 0, y: 8 } },
            rejectedInsertsPanel { gridPos: { h: 8, w: 12, x: 12, y: 8 } },
          ],
          //next row
          [
            memoryUsagePanel { gridPos: { h: 8, w: 12, x: 0, y: 16 } },
            memoryUsageGaugePanel { gridPos: { h: 8, w: 12, x: 12, y: 16 } },
          ],
          //next row
          [
            activeConnectionsPanel { gridPos: { h: 8, w: 24, x: 0, y: 24 } },
          ],
          //next row
          [
            networkReceivedPanel { gridPos: { h: 8, w: 12, x: 0, y: 32 } },
            networkTransmittedPanel { gridPos: { h: 8, w: 12, x: 12, y: 32 } },
          ],
          //next row
          if $._config.enableLokiLogs then [
            errorLogsPanel { gridPos: { h: 8, w: 24, x: 0, y: 40 } },
          ] else [],
        ])
      ),
  },
}
