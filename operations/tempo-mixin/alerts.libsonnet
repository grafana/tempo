{
  prometheusAlerts+:: {
    groups+: [
      {
        name: 'tempo_alerts',
        rules: [
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
            alert: 'TempoLiveStoreUnhealthy',
            'for': '15m',
            expr: |||
              max by (%s) (tempo_ring_members{state="Unhealthy", name="%s", namespace=~"%s"}) > 0
            ||| % [$._config.group_by_cluster, $._config.jobs.live_store, $._config.namespace],
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: 'There are {{ printf "%f" $value }} unhealthy livestore(s).',
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoLiveStoreUnhealthy',
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
              message: 'Tempo block list length is up 40 percent over the last 7 days. Consider scaling compactors.',
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
          // compactors
          {
            alert: 'TempoCompactionTooManyOutstandingBlocks',
            expr: |||
              sum by (%s) (tempodb_compaction_outstanding_blocks{container="compactor", namespace=~"%s"}) / ignoring(tenant) group_left count(tempo_build_info{container="compactor", namespace=~"%s"}) by (%s) > %d
            ||| % [$._config.group_by_tenant, $._config.namespace, $._config.namespace, $._config.group_by_cluster, $._config.alerts.outstanding_blocks_warning],
            'for': '6h',
            labels: {
              severity: 'warning',
            },
            annotations: {
              message: "There are too many outstanding compaction blocks in {{ $labels.%s }}/{{ $labels.namespace }} for tenant {{ $labels.tenant }}, increase compactor's CPU or add more compactors." % $._config.per_cluster_label,
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoCompactionTooManyOutstandingBlocks',
            },
          },
          {
            alert: 'TempoCompactionTooManyOutstandingBlocks',
            expr: |||
              sum by (%s) (tempodb_compaction_outstanding_blocks{container="compactor", namespace=~"%s"}) / ignoring(tenant) group_left count(tempo_build_info{container="compactor", namespace=~"%s"}) by (%s) > %d
            ||| % [$._config.group_by_tenant, $._config.namespace, $._config.namespace, $._config.group_by_cluster, $._config.alerts.outstanding_blocks_critical],
            'for': '24h',
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: "There are too many outstanding compaction blocks in {{ $labels.%s }}/{{ $labels.namespace }} for tenant {{ $labels.tenant }}, increase compactor's CPU or add more compactors." % $._config.per_cluster_label,
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoCompactionTooManyOutstandingBlocks',
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
              message: 'Tempo partition {{ $labels.partition }} is lagging by more than %d seconds in {{ $labels.%s }}/{{ $labels.namespace }}.' % [$._config.alerts.partition_lag_critical_seconds, $._config.per_cluster_label],
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
              message: 'Tempo ingest partition {{ $labels.partition }} for blockbuilder is lagging by more than %d seconds in {{ $labels.%s }}/{{ $labels.namespace }}.' % [$._config.alerts.block_builder_partition_lag_critical_seconds, $._config.per_cluster_label],
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
              message: 'Tempo ingest partition {{ $labels.partition }} for blockbuilder is lagging by more than %d seconds in {{ $labels.%s }}/{{ $labels.namespace }}.' % [$._config.alerts.block_builder_partition_lag_critical_seconds, $._config.per_cluster_label],
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoPartitionLag',
            },
          },
          {
            alert: 'TempoLiveStorePartitionLagWarning',
            expr: |||
              max by (%s, partition, group) (avg_over_time(tempo_ingest_group_partition_lag_seconds{namespace=~"%s", container="%s"}[6m])) > %d
            ||| % [$._config.group_by_cluster, $._config.namespace, $._config.jobs.live_store, $._config.alerts.live_store_partition_lag_warning_seconds],
            'for': '5m',
            labels: {
              severity: 'warning',
            },
            annotations: {
              message: 'Tempo ingest partition {{ $labels.partition }} for live store {{ $labels.group }} is lagging by more than %d seconds in {{ $labels.%s }}/{{ $labels.namespace }}.' % [$._config.alerts.live_store_partition_lag_critical_seconds, $._config.per_cluster_label],
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoPartitionLag',
            },
          },
          {
            alert: 'TempoLiveStorePartitionLagCritical',
            expr: |||
              max by (%s, partition, group) (avg_over_time(tempo_ingest_group_partition_lag_seconds{namespace=~"%s", container=~"%s"}[6m])) > %d
            ||| % [$._config.group_by_cluster, $._config.namespace, $._config.jobs.live_store, $._config.alerts.live_store_partition_lag_critical_seconds],
            'for': '5m',
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: 'Tempo ingest partition {{ $labels.partition }} for live store group {{ $labels.group }} is lagging by more than %d seconds in {{ $labels.%s }}/{{ $labels.namespace }}.' % [$._config.alerts.live_store_partition_lag_critical_seconds, $._config.per_cluster_label],
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

          {
            alert: 'TempoMemcachedErrorsElevated',
            expr: |||
              sum(rate(tempo_memcache_request_duration_seconds_count{status_code="500"}[5m])) by (cluster, namespace, name)
              /
              sum(rate(tempo_memcache_request_duration_seconds_count{}[5m])) by (cluster, namespace, name) > 0.2
            |||,
            'for': '10m',
            labels: {
              severity: 'warning',
            },
            annotations: {
              message: 'Tempo memcached error rate is {{ printf "%0.2f" $value }} for role {{ $labels.name }} in {{ $labels.cluster }}/{{ $labels.namespace }}.',
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoMemcachedErrorsElevated',
            },
          },

          {
            alert: 'TempoBlockBuildersPartitionsMismatch',
            expr: |||
              max(tempo_partition_ring_partitions{name=~"livestore-partitions", state=~"Active|Inactive"}) by (namespace,cluster)
              >
              sum(tempo_block_builder_owned_partitions) by(namespace,cluster)
            |||,
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: 'Tempo block-builder partitions mismatch in {{ $labels.cluster }}/{{ $labels.namespace }}.',
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoBlockBuildersPartitionsMismatch',
            },
            'for': '10m',
          },

          {
            alert: 'TempoLiveStoresPartitionsUnowned',
            expr: |||
              max by(namespace, cluster) (
                tempo_partition_ring_partitions{name=~"livestore-partitions", state=~"Active|Inactive"}
              )
              >
              count(count by (partition,namespace,cluster) (
                tempo_live_store_partition_owned{}
              )) by (namespace, cluster)
            |||,
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: 'Some live-store partitions are unowned in {{ $labels.cluster }}/{{ $labels.namespace }}.',
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoLiveStoresPartitionsUnowned',
            },
            'for': '10m',
          },
          // Zone severely degraded (60% down)
          {
            alert: 'TempoLiveStoreZoneSeverelyDegraded',
            expr: |||
              abs(
                (
                  count by (namespace, cluster, zone) (tempo_live_store_partition_owned)
                  /
                  on(namespace, cluster)
                  group_left()
                  max by (namespace, cluster) (count by (namespace, cluster, zone) (tempo_live_store_partition_owned))
                 ) - 1
               ) > 0.6
            |||,
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: 'Live-store zone {{ $labels.zone }} owns far fewer partitions than peers in {{ $labels.cluster }}/{{ $labels.namespace }}.',
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoLiveStoreZoneSeverelyDegraded',
            },
            'for': '10m',
          },


          {
            alert: 'TempoDistributorUsageTrackerErrors',
            expr: |||
              sum by (cluster, namespace, tenant, reason)(rate(tempo_distributor_usage_tracker_errors_total{}[5m])) > 0
            |||,
            'for': '30m',
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: 'Tempo distributor usage tracker errors for tenant {{ $labels.tenant }} in {{ $labels.cluster }}/{{ $labels.namespace }} (reason: {{ $labels.reason }}).',
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoDistributorUsageTrackerErrors',
            },
          },
          {
            alert: 'TempoMetricsGeneratorProcessorUpdatesFailing',
            expr: |||
              sum by (cluster, namespace, tenant) (
                increase(tempo_metrics_generator_active_processors_update_failed_total{namespace=~"%s"}[5m])
              ) > 0
            ||| % $._config.namespace,
            'for': '15m',
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: 'Metrics-generator processor updates are failing for tenant {{ $labels.tenant }} in {{ $labels.cluster }}/{{ $labels.namespace }}.',
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoMetricsGeneratorProcessorUpdatesFailing',
            },
          },
          {
            alert: 'TempoMetricsGeneratorServiceGraphsDroppingSpans',
            // 99.5
            expr: |||
              sum by (cluster, namespace, tenant) (increase(tempo_metrics_generator_processor_service_graphs_dropped_spans{namespace=~"%s"}[1h]))
              /
              sum by (cluster, namespace, tenant) (increase(tempo_metrics_generator_spans_received_total{namespace=~"%s"}[1h]))
              > 0.005
            ||| % [$._config.namespace, $._config.namespace],
            'for': '15m',
            labels: {
              severity: 'warning',
            },
            annotations: {
              message: 'Metrics-generator service-graphs processor is dropping {{ $value | humanizePercentage }} spans for tenant {{ $labels.tenant }} in {{ $labels.cluster }}/{{ $labels.namespace }}.',
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoMetricsGeneratorServiceGraphsDroppingSpans',
            },
          },
          {
            alert: 'TempoMetricsGeneratorCollectionsFailing',
            expr: |||
              sum by (cluster, namespace, tenant, pod, job) (increase(tempo_metrics_generator_registry_collections_failed_total{namespace=~"%s"}[5m])) > 2
            ||| % $._config.namespace,
            'for': '5m',
            labels: {
              severity: 'critical',
            },
            annotations: {
              message: 'Metrics-generator collections are failing for tenant {{ $labels.tenant }} in {{ $labels.cluster }}/{{ $labels.namespace }}.',
              runbook_url: 'https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/runbook.md#TempoMetricsGeneratorCollectionsFailing',
            },
          },
        ],
      },
    ],
  },
}
