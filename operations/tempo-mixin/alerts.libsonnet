{
  prometheusAlerts+:: {
    groups+: [
      {
        name: 'tempo_alerts',
        rules: [
          {
            alert: 'TempoRequestErrors',
            expr: |||
              100 * sum(rate(tempo_request_duration_seconds_count{status_code=~"5.."}[1m])) by (%(group_by_job)s, route)
                /
              sum(rate(tempo_request_duration_seconds_count[1m])) by (%(group_by_job)s, route)
                > 10
            ||| % $._config,
            'for': '15m',
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: |||
                {{ $labels.job }} {{ $labels.route }} is experiencing {{ printf "%.2f" $value }}% errors.
              |||,
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoRequestErrors',
            },
          },
          {
            alert: 'TempoRequestLatency',
            expr: |||
              %s_route:tempo_request_duration_seconds:99quantile{route!~"%s"} > %s
            ||| % [$._config.group_prefix_jobs, $._config.alerts.p99_request_exclude_regex, $._config.alerts.p99_request_threshold_seconds],
            'for': '15m',
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: |||
                {{ $labels.job }} {{ $labels.route }} is experiencing {{ printf "%.2f" $value }}s 99th percentile latency.
              |||,
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoRequestLatency',
            },
          },
          {
            alert: 'TempoCompactorUnhealthy',
            expr: |||
              max by (%s) (cortex_ring_members{state="Unhealthy", name="%s", namespace=~"%s"}) > 0
            ||| % [$._config.group_by_cluster, $._config.jobs.compactor, $._config.namespace],
            'for': '15m',
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: 'There are {{ printf "%f" $value }} unhealthy compactor(s).',
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoCompactorUnhealthy',
            },
          },
          {
            alert: 'TempoDistributorUnhealthy',
            'for': '15m',
            expr: |||
              max by (%s) (cortex_ring_members{state="Unhealthy", name="%s", namespace=~"%s"}) > 0
            ||| % [$._config.group_by_cluster, $._config.jobs.distributor, $._config.namespace],
            labels: {
              severity: 'warning',
            },
            annotations: {
              message: 'There are {{ printf "%f" $value }} unhealthy distributor(s).',
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoDistributorUnhealthy',
            },
          },
          {
            alert: 'TempoCompactionsFailing',
            'for': '5m',
            expr: |||
              sum by (%s) (increase(tempodb_compaction_errors_total{}[1h])) > %s and
              sum by (%s) (increase(tempodb_compaction_errors_total{}[5m])) > 0
            ||| % [$._config.group_by_cluster, $._config.alerts.compactions_per_hour_failed, $._config.group_by_cluster],
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: 'Greater than %s compactions have failed in the past hour.' % $._config.alerts.compactions_per_hour_failed,
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoCompactionsFailing',
            },
          },
          {
            // wait 5m for failed flushes to self-heal using retries
            alert: 'TempoIngesterFlushesFailing',
            expr: |||
              sum by (%s) (increase(tempo_ingester_failed_flushes_total{}[1h])) > %s and
              sum by (%s) (increase(tempo_ingester_failed_flushes_total{}[5m])) > 0
            ||| % [$._config.group_by_cluster, $._config.alerts.flushes_per_hour_failed, $._config.group_by_cluster],
            'for': '5m',
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: 'Greater than %s flushes have failed in the past hour.' % $._config.alerts.flushes_per_hour_failed,
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoIngesterFlushesFailing',
            },
          },
          {
            alert: 'TempoPollsFailing',
            expr: |||
              sum by (%s) (increase(tempodb_blocklist_poll_errors_total{}[1h])) > %s and
              sum by (%s) (increase(tempodb_blocklist_poll_errors_total{}[5m])) > 0
            ||| % [$._config.group_by_cluster, $._config.alerts.polls_per_hour_failed, $._config.group_by_cluster],
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: 'Greater than %s polls have failed in the past hour.' % $._config.alerts.polls_per_hour_failed,
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoPollsFailing',
            },
          },
          {
            alert: 'TempoTenantIndexFailures',
            expr: |||
              sum by (%s) (increase(tempodb_blocklist_tenant_index_errors_total{}[1h])) > %s and
              sum by (%s) (increase(tempodb_blocklist_tenant_index_errors_total{}[5m])) > 0
            ||| % [$._config.group_by_cluster, $._config.alerts.polls_per_hour_failed, $._config.group_by_cluster],
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: 'Greater than %s tenant index failures in the past hour.' % $._config.alerts.polls_per_hour_failed,
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoTenantIndexFailures',
            },
          },
          {
            alert: 'TempoNoTenantIndexBuilders',
            expr: |||
              sum by (%(group_by_tenant)s) (tempodb_blocklist_tenant_index_builder{}) == 0 and
              max by (%(group_by_cluster)s) (tempodb_blocklist_length{}) > 0
            ||| % $._config,
            'for': '5m',
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: 'No tenant index builders for tenant {{ $labels.tenant }}. Tenant index will quickly become stale.',
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoNoTenantIndexBuilders',
            },
          },
          {
            alert: 'TempoTenantIndexTooOld',
            expr: |||
              max by (%s) (tempodb_blocklist_tenant_index_age_seconds{}) > %s
            ||| % [$._config.group_by_tenant, $._config.alerts.max_tenant_index_age_seconds],
            'for': '5m',
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: 'Tenant index age is %s seconds old for tenant {{ $labels.tenant }}.' % $._config.alerts.max_tenant_index_age_seconds,
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoTenantIndexTooOld',
            },
          },
          {
            alert: 'TempoBadOverrides',
            expr: |||
              sum(tempo_runtime_config_last_reload_successful{namespace=~"%s"} == 0) by (cluster, namespace, job)
            ||| % $._config.namespace,
            'for': '15m',
            labels: {
              severity: 'warning',
            },
            annotations: {
              message: '{{ $labels.job }} failed to reload overrides.',
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoBadOverrides',
            },
          },
          // ingesters
          {
            alert: 'TempoProvisioningTooManyWrites',
            // 30MB/s written to the WAL per ingester max
            expr: |||
              avg by (cluster, namespace) (rate(tempo_ingester_bytes_received_total{job=~".+/ingester"}[1m])) / 1024 / 1024 > 30
            |||,
            'for': '15m',
            labels: {
              severity: 'warning',
            },
            annotations: {
              message: 'Ingesters in {{ $labels.cluster }}/{{ $labels.namespace }} are receiving more data/second than desired, add more ingesters.',
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoProvisioningTooManyWrites',
            },
          },
          // compactors
          {
            alert: 'TempoCompactorsTooManyOutstandingBlocks',
            expr: |||
              sum by (cluster, namespace, tenant) (tempodb_compaction_outstanding_blocks{container="compactor", namespace=~"%s"}) / ignoring(tenant) group_left count(tempo_build_info{container="compactor", namespace=~"%s"}) by (cluster, namespace) > %d
            ||| % [$._config.namespace, $._config.namespace, $._config.alerts.outstanding_blocks_warning],
            'for': '6h',
            labels: {
              severity: 'warning',
            },
            annotations: {
              message: "There are too many outstanding compaction blocks in {{ $labels.cluster }}/{{ $labels.namespace }} for tenant {{ $labels.tenant }}, increase compactor's CPU or add more compactors.",
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoCompactorsTooManyOutstandingBlocks',
            },
          },
          {
            alert: 'TempoCompactorsTooManyOutstandingBlocks',
            expr: |||
              sum by (cluster, namespace, tenant) (tempodb_compaction_outstanding_blocks{container="compactor", namespace=~"%s"}) / ignoring(tenant) group_left count(tempo_build_info{container="compactor", namespace=~"%s"}) by (cluster, namespace) > %d
            ||| % [$._config.namespace, $._config.namespace, $._config.alerts.outstanding_blocks_critical],
            'for': '24h',
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: "There are too many outstanding compaction blocks in {{ $labels.cluster }}/{{ $labels.namespace }} for tenant {{ $labels.tenant }}, increase compactor's CPU or add more compactors.",
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoCompactorsTooManyOutstandingBlocks',
            },
          },
        ],
      },
    ],
  },
}
