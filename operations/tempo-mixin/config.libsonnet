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
  },
}
