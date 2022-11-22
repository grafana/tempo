local g = (import 'grafana-builder/grafana.libsonnet');
local grafana = (import 'grafonnet/grafana.libsonnet');
local dashboard = grafana.dashboard;
local template = grafana.template;
local dashboardUid = 'clickhouse-latency';
local matcher = 'job=~"$job", instance=~"$instance"';

local diskReadLatencyPanel =
  {
    datasource: {
      type: 'prometheus',
      uid: '${prometheus_datasource}',
    },
    description: 'Time spent waiting for read syscall',
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
        unit: 'µs',
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
        expr: 'rate(ClickHouseProfileEvents_DiskReadElapsedMicroseconds{' + matcher + '}[$__rate_interval])',
        legendFormat: 'Disk Read Elapsed Microseconds',
        range: true,
        refId: 'A',
      },
    ],
    title: 'Disk Read Latency',
    type: 'timeseries',
  };

local diskWriteLatencyPanel =
  {
    datasource: {
      type: 'prometheus',
      uid: '${prometheus_datasource}',
    },
    description: 'Time spent waiting for write syscall',
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
        unit: 'µs',
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
        expr: 'rate(ClickHouseProfileEvents_DiskWriteElapsedMicroseconds{' + matcher + '}[$__rate_interval])',
        legendFormat: 'Disk Write Elapsed Microseconds',
        range: true,
        refId: 'A',
      },
    ],
    title: 'Disk Write Latency',
    type: 'timeseries',
  };

local networkTransmitLatencyPanel =
  {
    datasource: {
      type: 'prometheus',
      uid: '${prometheus_datasource}',
    },
    description: 'Latency of inbound network traffic',
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
        unit: 'µs',
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
        expr: 'rate(ClickHouseProfileEvents_NetworkReceiveElapsedMicroseconds{' + matcher + '}[$__rate_interval])',
        legendFormat: 'Network Receive Elapsed Microseconds',
        range: true,
        refId: 'A',
      },
    ],
    title: 'Network Receive Latency',
    type: 'timeseries',
  };

local networkTransmitLatencyPanel =
  {
    datasource: {
      type: 'prometheus',
      uid: '${prometheus_datasource}',
    },
    description: 'Latency of outbound network traffic',
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
        unit: 'µs',
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
        expr: 'rate(ClickHouseProfileEvents_NetworkSendElapsedMicroseconds{' + matcher + '}[$__rate_interval])',
        legendFormat: 'Network Send Elapsed Microseconds',
        range: true,
        refId: 'A',
      },
    ],
    title: 'Network Transmit Latency',
    type: 'timeseries',
  };

local zooKeeperWaitTimePanel =
  {
    datasource: {
      type: 'prometheus',
      uid: '${prometheus_datasource}',
    },
    description: 'Time spent waiting for ZooKeeper request to process',
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
        unit: 'µs',
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
        expr: 'rate(ClickHouseProfileEvents_ZooKeeperWaitMicroseconds{' + matcher + '}[$__rate_interval])',
        legendFormat: 'ZooKeeper Wait Microseconds',
        range: true,
        refId: 'A',
      },
    ],
    title: 'ZooKeeper Wait Time',
    type: 'timeseries',
  };
{
  grafanaDashboards+:: {

    'clickhouse-latency.json':
      dashboard.new(
        'Clickhouse Latency',
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
            query='label_values(ClickHouseProfileEvents_DiskReadElapsedMicroseconds{job=~"$job"}, instance)',
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
            diskReadLatencyPanel { gridPos: { h: 8, w: 12, x: 0, y: 0 } },
            diskWriteLatencyPanel { gridPos: { h: 8, w: 12, x: 12, y: 0 } },
          ],
          //next row
          [
            networkTransmitLatencyPanel { gridPos: { h: 8, w: 12, x: 0, y: 8 } },
            networkTransmitLatencyPanel { gridPos: { h: 8, w: 12, x: 12, y: 8 } },
          ],
          //next row
          [
            zooKeeperWaitTimePanel { gridPos: { h: 8, w: 24, x: 0, y: 16 } },
          ],
        ])
      ),
  },
}
