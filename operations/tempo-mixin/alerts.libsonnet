{
  prometheusAlerts+:: {
    groups+: [
      {
        name: 'tempo_alerts',
        rules: [
          {
            alert: 'TempoCompactorUnhealthy',
            expr: |||
              max by (%s) (tempo_ring_members{state="Unhealthy", name="%s", namespace=~"%s"}) > 0
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
              max by (%s) (tempo_ring_members{state="Unhealthy", name="%s", namespace=~"%s"}) > 0
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
            alert: 'TempoIngesterUnhealthy',
            'for': '15m',
            expr: |||
              max by (%s) (tempo_ring_members{state="Unhealthy", name="%s", namespace=~"%s"}) > 0
            ||| % [$._config.group_by_cluster, $._config.jobs.ingester, $._config.namespace],
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: 'There are {{ printf "%f" $value }} unhealthy ingester(s).',
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoIngesterUnhealthy',
            },
          },
          {
            alert: 'TempoMetricsGeneratorUnhealthy',
            'for': '15m',
            expr: |||
              max by (%s) (tempo_ring_members{state="Unhealthy", name="%s", namespace=~"%s"}) > 0
            ||| % [$._config.group_by_cluster, $._config.jobs.metrics_generator, $._config.namespace],
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: 'There are {{ printf "%f" $value }} unhealthy metric-generator(s).',
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoMetricsGeneratorUnhealthy',
            },
          },
          {
            alert: 'TempoCompactionsFailing',
            'for': '1h',
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
            alert: 'TempoIngesterFlushesUnhealthy',
            expr: |||
              sum by (%s) (increase(tempo_ingester_failed_flushes_total{}[1h])) > %s and
              sum by (%s) (increase(tempo_ingester_failed_flushes_total{}[5m])) > 0
            ||| % [$._config.group_by_cluster, $._config.alerts.flushes_per_hour_failed, $._config.group_by_cluster],
            'for': '5m',
            labels: {
              severity: 'warning',
            },
            annotations: {
              message: 'Greater than %s flush retries have occurred in the past hour.' % $._config.alerts.flushes_per_hour_failed,
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoIngesterFlushesFailing',
            },
          },
          {
            // wait 10m for failed flushes to self-heal using retries
            alert: 'TempoIngesterFlushesFailing',
            expr: |||
              sum by (%s) (increase(tempo_ingester_flush_failed_retries_total{}[1h])) > %s and
              sum by (%s) (increase(tempo_ingester_flush_failed_retries_total{}[5m])) > 0
            ||| % [$._config.group_by_cluster, $._config.alerts.flushes_per_hour_failed, $._config.group_by_cluster],
            'for': '5m',
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: 'Greater than %s flush retries have failed in the past hour.' % $._config.alerts.flushes_per_hour_failed,
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
            alert: 'TempoBlockListRisingQuickly',
            expr: |||
              avg(tempodb_blocklist_length{namespace=~"%(namespace)s", container="compactor"}) / avg(tempodb_blocklist_length{namespace=~"%(namespace)s", container="compactor"} offset 7d) by (%(group)s) > 1.4
            ||| % { namespace: $._config.namespace, group: $._config.group_by_cluster },
            'for': '15m',
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: 'Tempo block list length is up 40 percent over the last 7 days.  Consider scaling compactors.',
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoBlockListRisingQuickly',
            },
          },
          {
            alert: 'TempoBadOverrides',
            expr: |||
              sum(tempo_runtime_config_last_reload_successful{namespace=~"%s"} == 0) by (%s)
            ||| % [$._config.namespace, $._config.group_by_job],
            'for': '15m',
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: '{{ $labels.job }} failed to reload overrides.',
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoBadOverrides',
            },
          },
          {
            alert: 'TempoUserConfigurableOverridesReloadFailing',
            expr: |||
              sum by (%s) (increase(tempo_overrides_user_configurable_overrides_reload_failed_total{}[1h])) > %s and
              sum by (%s) (increase(tempo_overrides_user_configurable_overrides_reload_failed_total{}[5m])) > 0
            ||| % [$._config.group_by_cluster, $._config.alerts.user_configurable_overrides_polls_per_hour_failed, $._config.group_by_cluster],
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: 'Greater than %s user-configurable overides reloads failed in the past hour.' % $._config.alerts.user_configurable_overrides_polls_per_hour_failed,
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoTenantIndexFailures',
            },
          },
          // ingesters
          {
            alert: 'TempoProvisioningTooManyWrites',
            // 30MB/s written to the WAL per ingester max
            expr: |||
              avg by (%s) (rate(tempo_ingester_bytes_received_total{job=~".+/ingester"}[5m])) / 1024 / 1024 > 30
            ||| % $._config.group_by_cluster,
            'for': '15m',
            labels: {
              severity: 'warning',
            },
            annotations: {
              message: 'Ingesters in {{ $labels.%s }}/{{ $labels.namespace }} are receiving more data/second than desired, add more ingesters.' % $._config.per_cluster_label,
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoProvisioningTooManyWrites',
            },
          },
          // compactors
          {
            alert: 'TempoCompactorsTooManyOutstandingBlocks',
            expr: |||
              sum by (%s) (tempodb_compaction_outstanding_blocks{container="compactor", namespace=~"%s"}) / ignoring(tenant) group_left count(tempo_build_info{container="compactor", namespace=~"%s"}) by (%s) > %d
            ||| % [$._config.group_by_tenant, $._config.namespace, $._config.namespace, $._config.group_by_cluster, $._config.alerts.outstanding_blocks_warning],
            'for': '6h',
            labels: {
              severity: 'warning',
            },
            annotations: {
              message: "There are too many outstanding compaction blocks in {{ $labels.%s }}/{{ $labels.namespace }} for tenant {{ $labels.tenant }}, increase compactor's CPU or add more compactors." % $._config.per_cluster_label,
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoCompactorsTooManyOutstandingBlocks',
            },
          },
          {
            alert: 'TempoCompactorsTooManyOutstandingBlocks',
            expr: |||
              sum by (%s) (tempodb_compaction_outstanding_blocks{container="compactor", namespace=~"%s"}) / ignoring(tenant) group_left count(tempo_build_info{container="compactor", namespace=~"%s"}) by (%s) > %d
            ||| % [$._config.group_by_tenant, $._config.namespace, $._config.namespace, $._config.group_by_cluster, $._config.alerts.outstanding_blocks_critical],
            'for': '24h',
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: "There are too many outstanding compaction blocks in {{ $labels.%s }}/{{ $labels.namespace }} for tenant {{ $labels.tenant }}, increase compactor's CPU or add more compactors." % $._config.per_cluster_label,
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoCompactorsTooManyOutstandingBlocks',
            },
          },
          {
            alert: 'TempoIngesterReplayErrors',
            'for': '5m',
            expr: |||
              sum by (%s) (increase(tempo_ingester_replay_errors_total{namespace=~"%s"}[5m])) > 0
            ||| % [$._config.group_by_tenant, $._config.namespace],
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: 'Tempo ingester has encountered errors while replaying a block on startup in {{ $labels.%s }}/{{ $labels.namespace }} for tenant {{ $labels.tenant }}' % $._config.per_cluster_label,
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoIngesterReplayErrors',
            },
          },
          {
            alert: 'TempoMetricsGeneratorPartitionLagCritical',
            expr: |||
              max by (%s, partition) (tempo_ingest_group_partition_lag_seconds{namespace=~"%s", container="%s"}) > %d
            ||| % [$._config.group_by_cluster, $._config.namespace, $._config.jobs.metrics_generator, $._config.alerts.partition_lag_critical_seconds],
            'for': '5m',
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: 'Tempo partition {{ $labels.partition }} in consumer group {{ $labels.group }} is lagging by more than %d seconds in {{ $labels.%s }}/{{ $labels.namespace }}.' % [$._config.alerts.partition_lag_critical_seconds, $._config.per_cluster_label],
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoPartitionLag',
            },
          },
          {
            alert: 'TempoBlockBuilderPartitionLagWarning',
            expr: |||
              max by (%s, partition) (avg_over_time(tempo_ingest_group_partition_lag_seconds{namespace=~"%s", container="%s"}[6m])) > %d
            ||| % [$._config.group_by_cluster, $._config.namespace, $._config.jobs.block_builder, $._config.alerts.block_builder_partition_lag_warning_seconds],
            'for': '5m',
            labels: {
              severity: 'warning',
            },
            annotations: {
              message: 'Tempo ingest partition {{ $labels.partition }} for blockbuilder {{ $labels.pod }} is lagging by more than %d seconds in {{ $labels.%s }}/{{ $labels.namespace }}.' % [$._config.alerts.block_builder_partition_lag_critical_seconds, $._config.per_cluster_label],
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoPartitionLag',
            },
          },
          {
            alert: 'TempoBlockBuilderPartitionLagCritical',
            expr: |||
              max by (%s, partition) (avg_over_time(tempo_ingest_group_partition_lag_seconds{namespace=~"%s", container=~"%s"}[6m])) > %d
            ||| % [$._config.group_by_cluster, $._config.namespace, $._config.jobs.block_builder, $._config.alerts.block_builder_partition_lag_critical_seconds],
            'for': '5m',
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: 'Tempo ingest partition {{ $labels.partition }} for blockbuilder {{ $labels.pod }} is lagging by more than %d seconds in {{ $labels.%s }}/{{ $labels.namespace }}.' % [$._config.alerts.block_builder_partition_lag_critical_seconds, $._config.per_cluster_label],
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoPartitionLag',
            },
          },
          {
            alert: 'TempoBackendSchedulerJobsFailureRateHigh',
            expr: |||
              sum(increase(tempo_backend_scheduler_jobs_failed_total{namespace=~"%s"}[5m])) by (%s)
              /
              sum(increase(tempo_backend_scheduler_jobs_created_total{namespace=~"%s"}[5m])) by (%s)
              > %0.2f
            ||| % [$._config.namespace, $._config.group_by_cluster, $._config.namespace, $._config.group_by_cluster, $._config.alerts.backend_scheduler_jobs_failure_rate],
            'for': '10m',
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: 'Tempo backend scheduler job failure rate is {{ printf "%0.2f" $value }} (threshold 0.1) in {{ $labels.cluster }}/{{ $labels.namespace }}',
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoBackendSchedulerJobsFailureRateHigh',
            },
          },
          {
            alert: 'TempoBackendSchedulerRetryRateHigh',
            expr: |||
              sum(increase(tempo_backend_scheduler_jobs_retry_total{namespace=~"%s"}[1m])) by (%s) > %s
            ||| % [$._config.namespace, $._config.group_by_cluster, $._config.alerts.backend_scheduler_jobs_retry_count_per_minute],
            'for': '10m',
            labels: {
              severity: 'warning',
            },
            annotations: {
              message: 'Tempo backend scheduler retry rate is high ({{ printf "%0.2f" $value }} retries/minute) in {{ $labels.cluster }}/{{ $labels.namespace }}',
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoBackendSchedulerRetryRateHigh',
            },
          },
          {
            alert: 'TempoBackendSchedulerCompactionEmptyJobRateHigh',
            expr: |||
              sum(increase(tempo_backend_scheduler_compaction_tenant_empty_job_total{namespace=~"%s"}[1m])) by (%s) > %s
            ||| % [$._config.namespace, $._config.group_by_cluster, $._config.alerts.backend_scheduler_compaction_tenant_empty_job_count_per_minute],
            'for': '10m',
            labels: {
              severity: 'warning',
            },
            annotations: {
              message: 'Tempo backend scheduler empty job rate is high ({{ printf "%0.2f" $value }} jobs/minute) in {{ $labels.cluster }}/{{ $labels.namespace }}',
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoBackendSchedulerCompactionEmptyJobRateHigh',
            },
          },
          {
            alert: 'TempoBackendWorkerBadJobsRateHigh',
            expr: |||
              sum(increase(tempo_backend_worker_bad_jobs_received_total{namespace=~"%s"}[1m])) by (%s) > %s
            ||| % [$._config.namespace, $._config.group_by_cluster, $._config.alerts.backend_scheduler_bad_jobs_count_per_minute],
            'for': '10m',
            labels: {
              severity: 'warning',
            },
            annotations: {
              message: 'Tempo backend worker bad jobs rate is high ({{ printf "%0.2f" $value }} bad jobs/minute) in {{ $labels.cluster }}/{{ $labels.namespace }}',
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoBackendWorkerBadJobsRateHigh',
            },
          },
          {
            alert: 'TempoBackendWorkerCallRetriesHigh',
            expr: |||
              sum(increase(tempo_backend_worker_call_retries_total{namespace=~"%s"}[1m])) by (%s) > %s
            ||| % [$._config.namespace, $._config.group_by_cluster, $._config.alerts.backend_worker_call_retries_count_per_minute],
            'for': '10m',
            labels: {
              severity: 'warning',
            },
            annotations: {
              message: 'Tempo backend worker call retries rate is high ({{ printf "%0.2f" $value }} retries/minute) in {{ $labels.cluster }}/{{ $labels.namespace }}',
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoBackendWorkerCallRetriesHigh',
            },
          },
          {
            alert: 'TempoVultureHighErrorRate',
            expr: |||
              sum(rate(tempo_vulture_trace_error_total{namespace=~"%s"}[1m])) by (%s, error) / ignoring (error) group_left sum(rate(tempo_vulture_trace_total{namespace=~"%s"}[1m])) by (%s) > %f
            ||| % [$._config.namespace, $._config.group_by_cluster, $._config.namespace, $._config.group_by_cluster, $._config.alerts.vulture_error_rate_threshold],
            'for': '5m',
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: 'Tempo vulture error rate is {{ printf "%0.2f" $value }} for error type {{ $labels.error }} in {{ $labels.cluster }}/{{ $labels.namespace }}',
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoVultureHighErrorRate',
            },
          },
        ],
      },
    ],
  },
}
