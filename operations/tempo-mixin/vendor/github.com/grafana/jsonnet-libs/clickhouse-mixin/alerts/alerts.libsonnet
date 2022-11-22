{
  prometheusAlerts+:: {
    groups+: [
      {
        name: 'ClickhouseAlerts',
        rules: [
          {
            alert: 'ClickhouseReplicationQueueBackingUp',
            expr: |||
              ClickHouseAsyncMetrics_ReplicasMaxQueueSize > %(alertsReplicasMaxQueueSize)s
            ||| % $._config,
            labels: {
              severity: 'warning',
            },
            annotations: {
              summary: 'Clickhouse replica max queue size backing up.',
              description: |||
                Clickhouse replication tasks are processing slower than expected on {{ $labels.instance }} causing replication queue size to back up at {{ $value }} exceeding the threshold value of %(alertsReplicasMaxQueueSize)s.
              ||| % $._config,
            },
            'for': '5m',
          },
          {
            alert: 'ClickhouseRejectedInserts',
            expr: 'ClickHouseProfileEvents_RejectedInserts > 1',
            labels: {
              severity: 'critical',
            },
            annotations: {
              summary: 'Clickhouse has too many rejected inserts.',
              description: 'Clickhouse inserts are being rejected on {{ $labels.instance }} as items are being inserted faster than Clickhouse is able to merge them.',
            },
            'for': '5m',
          },
          {
            alert: 'ClickhouseZookeeperSessions',
            expr: 'ClickHouseMetrics_ZooKeeperSession > 1',
            labels: {
              severity: 'critical',
            },
            annotations: {
              summary: 'Clickhouse has too many Zookeeper sessions.',
              description: |||
                Clickhouse has more than one connection to a Zookeeper on {{ $labels.instance }} which can lead to bugs due to stale reads in Zookeepers consistency model.
              |||,
            },
            'for': '5m',
          },
          {
            alert: 'ClickhouseReplicasInReadOnly',
            expr: 'ClickHouseMetrics_ReadonlyReplica > 0',
            labels: {
              severity: 'critical',
            },
            annotations: {
              summary: 'Clickhouse has too many replicas in read only state.',
              description: |||
                Clickhouse has replicas in a read only state on {{ $labels.instance }} after losing connection to Zookeeper or at startup.
              |||,
            },
            'for': '5m',
          },
        ],
      },
    ],
  },
}
