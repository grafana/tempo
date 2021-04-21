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
      tempoPushUrl: 'http://distributor:14250',
      tempoQueryUrl: 'http://query-frontend:3100',
      tempoOrgId: '',
    },
    ballast_size_mbs: '1024',
    port: 3100,
    http_api_prefix: '',
    gossip_ring_port: 7946,
    backend: error 'Must specify a backend',  // gcs|s3
    bucket: error 'Must specify a bucket',

    overrides+:: {
      super_user:: {
        max_traces_per_user: 100000,
        ingestion_rate_limit_bytes: 200e5,  // ~20MB per sec
        ingestion_burst_size_bytes: 200e5,  // ~20MB
        max_bytes_per_trace: 300e5,  // ~30MB
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
