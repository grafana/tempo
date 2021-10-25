{
  _images+:: {
    tempo: 'grafana/tempo:latest',
    tempo_query: 'grafana/tempo-query:latest',
    tempo_vulture: 'grafana/tempo-vulture:latest',
    memcached: 'memcached:1.6.9-alpine',
    memcachedExporter: 'prom/memcached-exporter:v0.6.0',
  },

  _config+:: {
    gossip_member_label: 'tempo-gossip-member',
    compactor: {
      replicas: 1,
      resources: {
        requests: {
          cpu: '500m',
          memory: '3Gi',
        },
        limits: {
          cpu: '1',
          memory: '5Gi',
        },
      },
    },
    query_frontend: {
      replicas: 1,
      resources: {
        requests: {
          cpu: '500m',
          memory: '1Gi',
        },
        limits: {
          cpu: '1',
          memory: '2Gi',
        },
      },
    },
    querier: {
      replicas: 2,
      resources: {
        requests: {
          cpu: '500m',
          memory: '1Gi',
        },
        limits: {
          cpu: '1',
          memory: '2Gi',
        },
      },
    },
    ingester: {
      pvc_size: error 'Must specify an ingester pvc size',
      pvc_storage_class: error 'Must specify an ingester pvc storage class',
      replicas: 3,
      resources: {
        requests: {
          cpu: '3',
          memory: '3Gi',
        },
        limits: {
          cpu: '5',
          memory: '5Gi',
        },
      },
    },
    distributor: {
      receivers: error 'Must specify receivers',
      replicas: 1,
      resources: {
        requests: {
          cpu: '3',
          memory: '3Gi',
        },
        limits: {
          cpu: '5',
          memory: '5Gi',
        },
      },
    },
    memcached: {
      replicas: 3,
      connection_limit: 4096,
      memory_limit_mb: 1024,
    },
    jaeger_ui: {
      base_path: '/',
    },
    vulture: {
      replicas: 0,
      tempoPushUrl: 'http://distributor',
      tempoQueryUrl: 'http://query-frontend:%s' % $._config.port,
      tempoOrgId: '',
      tempoRetentionDuration: '',
      tempoSearchBackoffDuration: '',
      tempoReadBackoffDuration: '',
      tempoWriteBackoffDuration: '',
    },
    ballast_size_mbs: '1024',
    port: 3200,
    http_api_prefix: '',
    gossip_ring_port: 7946,
    backend: error 'Must specify a backend',  // gcs|s3
    bucket: error 'Must specify a bucket',

    overrides_configmap_name: 'tempo-overrides',
    overrides+:: {
      super_user:: {
        max_traces_per_user: 100000,
        ingestion_rate_limit_bytes: 200e5,  // ~20MB per sec
        ingestion_burst_size_bytes: 200e5,  // ~20MB
        max_bytes_per_trace: 300e5,  // ~30MB
      },
    },
  },
}
