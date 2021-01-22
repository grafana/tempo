{
  _images+:: {
    tempo: 'grafana/tempo:latest',
    tempo_query: 'grafana/tempo-query:latest',
    tempo_vulture: 'grafana/tempo-vulture:latest',
    memcached: 'memcached:1.5.17-alpine',
    memcachedExporter: 'prom/memcached-exporter:v0.6.0',
  },

  _config+:: {
    gossip_member_label: 'tempo-gossip-member',
    compactor: {
      replicas: 1,
    },
    query_frontend: {
      replicas: 1,
    },
    querier: {
      replicas: 2,
    },
    ingester: {
      pvc_size: error 'Must specify an ingester pvc size',
      pvc_storage_class: error 'Must specify an ingester pvc storage class',
      replicas: 3,
    },
    distributor: {
      receivers: error 'Must specify receivers',
      replicas: 1,
    },
    memcached: {
      replicas: 3,
    },
    jaeger_ui: {
      base_path: '/',
    },
    vulture: {
      replicas: 0,
      loki_user: '',
      loki_pass: '',
      loki_base_url: '',
      loki_query: '',
    },
    ballast_size_mbs: '1024',
    port: 3100,
    gossip_ring_port: 7946,
    backend: error 'Must specify a backend',  // gcs|s3
    bucket: error 'Must specify a bucket',

    overrides+:: {
      super_user:: {
        max_traces_per_user: 100000,
        ingestion_rate_limit: 150000,
        ingestion_burst_size: 150000,
        max_spans_per_trace: 200e3,
      },
    },
  },

  tempo_compactor_container+::
    $.util.resourcesRequests('500m', '3Gi') +
    $.util.resourcesLimits('1', '5Gi'),

  tempo_distributor_container+::
    $.util.resourcesRequests('3', '3Gi') +
    $.util.resourcesLimits('5', '5Gi'),

  tempo_ingester_container+::
    $.util.resourcesRequests('3', '3Gi') +
    $.util.resourcesLimits('5', '5Gi'),

  tempo_query_frontend_container+::
    $.util.resourcesRequests('500m', '1Gi') +
    $.util.resourcesLimits('1', '2Gi'),

  tempo_querier_container+::
    $.util.resourcesRequests('500m', '1Gi') +
    $.util.resourcesLimits('1', '2Gi'),
}
