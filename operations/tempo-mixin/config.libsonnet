{
  local makePrefix(groups) = std.join('_', groups),
  local makeGroupBy(groups) = std.join(', ', groups),

  _config+:: {
    http_api_prefix: '',
    namespace: '.*',
    jobs: {
      gateway: 'cortex-gw(-internal)?',
      query_frontend: 'query-frontend',
      querier: 'querier',
      ingester: 'ingester',
      metrics_generator: 'metrics-generator',
      distributor: 'distributor',
      compactor: 'compactor',
      block_builder: 'block-builder',
      backend_scheduler: 'backend-scheduler',
      backend_worker: 'backend-worker',
      memcached: 'memcached',
    },
    alerts: {
      compactions_per_hour_failed: 2,
      flushes_per_hour_failed: 2,
      polls_per_hour_failed: 2,
      user_configurable_overrides_polls_per_hour_failed: 5,
      max_tenant_index_age_seconds: 600,
      p99_request_threshold_seconds: 3,
      p99_request_exclude_regex: 'metrics|/frontend.Frontend/Process|debug_pprof',
      outstanding_blocks_warning: 100,
      outstanding_blocks_critical: 250,
      // Generators partition lag thresholds in seconds
      partition_lag_critical_seconds: 900,  // 15 minutes
      // Block-builder partition lag thresholds in seconds
      block_builder_partition_lag_warning_seconds: 200,  // 3.3 minutes
      block_builder_partition_lag_critical_seconds: 300,  // 5 minutes
      // threshold config for backend scheduler and worker alerts
      backend_scheduler_jobs_failure_rate: 0.05,  // 5% of the jobs failed
      backend_scheduler_jobs_retry_count_per_minute: 20,  // 20 jobs retried per minute
      backend_scheduler_compaction_tenant_empty_job_count_per_minute: 10,  // 10 empty jobs per minute
      backend_scheduler_bad_jobs_count_per_minute: 0,  // alert if there are any bad jobs
      backend_worker_call_retries_count_per_minute: 5,  // 5 retries per minute
      vulture_error_rate_threshold: 0.1,  // 10% error rate
    },

    per_cluster_label: 'cluster',
    namespace_selector_separator: '/',

    // Groups labels to uniquely identify and group by {jobs, clusters, tenants}
    cluster_selectors: [$._config.per_cluster_label, 'namespace'],
    job_selectors: [$._config.per_cluster_label, 'namespace', 'job'],
    tenant_selectors: [$._config.per_cluster_label, 'namespace', 'tenant'],

    // Each group prefix is composed of `_`-separated labels
    group_prefix_clusters: makePrefix($._config.cluster_selectors),
    group_prefix_jobs: makePrefix($._config.job_selectors),
    group_prefix_tenants: makePrefix($._config.tenant_selectors),

    // Each group-by label list is `, `-separated and unique identifies
    group_by_cluster: makeGroupBy($._config.cluster_selectors),
    group_by_job: makeGroupBy($._config.job_selectors),
    group_by_tenant: makeGroupBy($._config.tenant_selectors),

    // Tunes histogram recording rules to aggregate over this interval.
    // Set to at least twice the scrape interval; otherwise, recording rules will output no data.
    // Set to four times the scrape interval to account for edge cases: https://www.robustperception.io/what-range-should-i-use-with-rate/
    recording_rules_range_interval: '1m',

  },
}
