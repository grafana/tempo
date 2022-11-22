local g = (import 'grafana-builder/grafana.libsonnet');
local grafana = (import 'grafonnet/grafana.libsonnet');
local dashboard = grafana.dashboard;
local template = grafana.template;
local dashboardUid = 'clickhouse-replica';
local matcher = 'job=~"$job", instance=~"$instance"';

local interserverConnectionsPanel =
  {
    datasource: {
      type: 'prometheus',
      uid: '${prometheus_datasource}',
    },
    description: 'Number of connections due to interserver communication',
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
        editorMode: 'code',
        expr: 'ClickHouseMetrics_InterserverConnection{' + matcher + '}',
        legendFormat: 'Interserver Connections',
        range: true,
        refId: 'A',
      },
    ],
    title: 'Interserver Connections',
    type: 'timeseries',
  };
local replicaQueueSizePanel =
  {
    datasource: {
      type: 'prometheus',
      uid: '${prometheus_datasource}',
    },
    description: 'Number of replica tasks in queue',
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
        editorMode: 'code',
        expr: 'ClickHouseAsyncMetrics_ReplicasMaxQueueSize{' + matcher + '}',
        legendFormat: 'Max Queue Size',
        range: true,
        refId: 'A',
      },
    ],
    title: 'Replica Queue Size',
    type: 'timeseries',
  };
local replicaOperationsPanel =
  {
    datasource: {
      type: 'prometheus',
      uid: '${prometheus_datasource}',
    },
    description: 'Replica Operations over time to other nodes',
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
        editorMode: 'code',
        expr: 'rate(ClickHouseProfileEvents_ReplicatedPartFetches{' + matcher + '}[$__rate_interval])',
        legendFormat: 'Part Fetches',
        range: true,
        refId: 'A',
      },
      {
        datasource: {
          type: 'prometheus',
          uid: '${prometheus_datasource}',
        },
        editorMode: 'code',
        expr: 'rate(ClickHouseProfileEvents_ReplicatedPartMerges{' + matcher + '}[$__rate_interval])',
        hide: false,
        legendFormat: 'Part Merges',
        range: true,
        refId: 'B',
      },
      {
        datasource: {
          type: 'prometheus',
          uid: '${prometheus_datasource}',
        },
        editorMode: 'code',
        expr: 'rate(ClickHouseProfileEvents_ReplicatedPartMutations{' + matcher + '}[$__rate_interval])',
        hide: false,
        legendFormat: 'Part Mutations',
        range: true,
        refId: 'C',
      },
      {
        datasource: {
          type: 'prometheus',
          uid: '${prometheus_datasource}',
        },
        editorMode: 'code',
        expr: 'rate(ClickHouseProfileEvents_ReplicatedPartChecks{' + matcher + '}[$__rate_interval])',
        hide: false,
        legendFormat: 'Part Checks',
        range: true,
        refId: 'D',
      },
    ],
    title: 'Replica Operations',
    type: 'timeseries',
  };
local replicaReadOnlyPanel =
  {
    datasource: {
      type: 'prometheus',
      uid: '${prometheus_datasource}',
    },
    description: 'Shows replicas in read-only state over time',
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
        editorMode: 'code',
        expr: 'ClickHouseMetrics_ReadonlyReplica{' + matcher + '}',
        legendFormat: 'Read Only',
        range: true,
        refId: 'A',
      },
    ],
    title: 'Replica Read Only',
    type: 'timeseries',
  };
local zooKeeperWatchesPanel =
  {
    datasource: {
      type: 'prometheus',
      uid: '${prometheus_datasource}',
    },
    description: 'Current number of watches in ZooKeeper',
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
        editorMode: 'code',
        expr: 'ClickHouseMetrics_ZooKeeperWatch{' + matcher + '}',
        legendFormat: 'ZooKeeper Watch',
        range: true,
        refId: 'A',
      },
    ],
    title: 'ZooKeeper Watches',
    type: 'timeseries',
  };
local zooKeeperSessionsPanel =
  {
    datasource: {
      type: 'prometheus',
      uid: '${prometheus_datasource}',
    },
    description: 'Current number of sessions to ZooKeeper',
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
        editorMode: 'code',
        expr: 'ClickHouseMetrics_ZooKeeperSession{' + matcher + '}',
        legendFormat: 'ZooKeeper Session',
        range: true,
        refId: 'A',
      },
    ],
    title: 'ZooKeeper Sessions',
    type: 'timeseries',
  };
local zooKeeperRequestsPanel =
  {
    datasource: {
      type: 'prometheus',
      uid: '${prometheus_datasource}',
    },
    description: 'Current number of active requests to ZooKeeper',
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
        editorMode: 'code',
        expr: 'ClickHouseMetrics_ZooKeeperRequest{' + matcher + '}',
        legendFormat: 'ZooKeeper Request',
        range: true,
        refId: 'A',
      },
    ],
    title: 'ZooKeeper Requests',
    type: 'timeseries',
  };
{
  grafanaDashboards+:: {

    'clickhouse-replica.json':
      dashboard.new(
        'Clickhouse Replica',
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
            query='label_values(ClickHouseMetrics_InterserverConnection{job=~"$job"}, instance)',
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
            interserverConnectionsPanel { gridPos: { h: 8, w: 12, x: 0, y: 0 } },
            replicaQueueSizePanel { gridPos: { h: 8, w: 12, x: 12, y: 0 } },
          ],
          //next row
          [
            replicaOperationsPanel { gridPos: { h: 8, w: 12, x: 0, y: 8 } },
            replicaReadOnlyPanel { gridPos: { h: 8, w: 12, x: 12, y: 8 } },
          ],
          //next row
          [
            zooKeeperWatchesPanel { gridPos: { h: 8, w: 12, x: 0, y: 16 } },
            zooKeeperSessionsPanel { gridPos: { h: 8, w: 12, x: 12, y: 16 } },
          ],
          //next row
          [
            zooKeeperRequestsPanel { gridPos: { h: 8, w: 24, x: 0, y: 24 } },
          ],
        ])
      ),
  },
}
