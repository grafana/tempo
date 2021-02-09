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
              namespace_job_route:tempo_request_duration_seconds:99quantile{route!~"metrics|/frontend.Frontend/Process"} > 3
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
              severity: 'warning',
            },
            annotations: {
              message: 'There are {{ printf "%f" $value }} unhealthy distributor(s).',
              runbook_url: 'https://github.com/grafana/tempo/tree/master/operations/tempo-mixin/runbook.md#TempoDistributorUnhealthy'
            },
          },
          {
            alert: 'TempoCompactionsFailing',
            expr: |||
              sum by (cluster, namespace) (increase(tempodb_compaction_errors_total{}[1h])) > %s and
              sum by (cluster, namespace) (increase(tempodb_compaction_errors_total{}[5m])) > 0
            ||| % $._config.alerts.compactions_per_hour_failed,
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: 'Greater than %s compactions have failed in the past hour.' % $._config.alerts.compactions_per_hour_failed,
              runbook_url: 'https://github.com/grafana/tempo/tree/master/operations/tempo-mixin/runbook.md#TempoCompactionsFailing'
            },
          },
          {
            alert: 'TempoFlushesFailing',
            expr: |||
              sum by (cluster, namespace) (increase(tempo_ingester_failed_flushes_total{}[1h])) > %s and
              sum by (cluster, namespace) (increase(tempo_ingester_failed_flushes_total{}[5m])) > 0
            ||| % $._config.alerts.flushes_per_hour_failed,
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: 'Greater than %s flushes have failed in the past hour.' % $._config.alerts.flushes_per_hour_failed,
              runbook_url: 'https://github.com/grafana/tempo/tree/master/operations/tempo-mixin/runbook.md#TempoFlushesFailing'
            },
          },
          {
            alert: 'TempoPollsFailing',
            expr: |||
              sum by (cluster, namespace) (increase(tempodb_blocklist_poll_errors_total{}[1h])) > %s and
              sum by (cluster, namespace) (increase(tempodb_blocklist_poll_errors_total{}[5m])) > 0
            ||| % $._config.alerts.polls_per_hour_failed,
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: 'Greater than %s polls have failed in the past hour.' % $._config.alerts.polls_per_hour_failed,
              runbook_url: 'https://github.com/grafana/tempo/tree/master/operations/tempo-mixin/runbook.md#TempoPollsFailing'
            },
          },
        ],
      },
    ],
  },
}
