{
  prometheusAlerts+:: {
    groups+: [
      {
        name: 'tempo_alerts',
        rules: [
          {
            alert: 'TempoRequestErrors',
            expr: |||
              100 * sum(rate(tempo_request_duration_seconds_count{status_code=~"5.."}[1m])) by (namespace, job, route)
                /
              sum(rate(tempo_request_duration_seconds_count[1m])) by (namespace, job, route)
                > 10
            |||,
            'for': '15m',
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: |||
                {{ $labels.job }} {{ $labels.route }} is experiencing {{ printf "%.2f" $value }}% errors.
              |||,
              runbook_url: 'https://github.com/grafana/tempo/tree/master/operations/tempo-mixin/runbook.md#TempoRequestErrors'
            },
          },
          {
            alert: 'TempoRequestLatency',
            expr: |||
              namespace_job_route:tempo_request_duration_seconds:99quantile{} > 3
            |||,
            'for': '15m',
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: |||
                {{ $labels.job }} {{ $labels.route }} is experiencing {{ printf "%.2f" $value }}s 99th percentile latency.
              |||,
              runbook_url: 'https://github.com/grafana/tempo/tree/master/operations/tempo-mixin/runbook.md#TempoRequestLatency'
            },
          },
          {
            alert: 'TempoCompactorUnhealthy',
            'for': '15m',
            expr: |||
              max by (cluster, namespace) (cortex_ring_members{state="Unhealthy", name="compactor"}) > 0
            |||,
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: 'There are {{ printf "%f" $value }} unhealthy compactor(s).',
              runbook_url: 'https://github.com/grafana/tempo/tree/master/operations/tempo-mixin/runbook.md#TempoCompactorUnhealthy'
            },
          },
          {
            alert: 'TempoDistributorUnhealthy',
            'for': '15m',
            expr: |||
              max by (cluster, namespace) (cortex_ring_members{state="Unhealthy", name="distributor"}) > 0
            |||,
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: 'There are {{ printf "%f" $value }} unhealthy distributor(s).',
              runbook_url: 'https://github.com/grafana/tempo/tree/master/operations/tempo-mixin/runbook.md#TempoDistributorUnhealthy'
            },
          },
        ],
      },
    ],
  },
}
