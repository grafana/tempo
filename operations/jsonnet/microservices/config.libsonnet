{
  _images+:: {
    tempo: 'grafana/tempo:latest',
    tempo_query: 'grafana/tempo-query:latest',
    tempo_vulture: 'grafana/tempo-vulture:latest',
    rollout_operator: 'grafana/rollout-operator:v0.23.0',
    memcached: 'memcached:1.6.38-alpine',
    memcachedExporter: 'prom/memcached-exporter:v0.15.2',
  },

  _config+:: {
    gossip_member_label: 'tempo-gossip-member',
    // Labels that service selectors should not use
    service_ignored_labels:: [self.gossip_member_label],

    variables_expansion: false,
    variables_expansion_env_mixin: null,
    node_selector: null,
    ingester_allow_multiple_replicas_on_same_node: false,

    // Enable concurrent rollout of block-builder through the usage of the rollout operator.
    // This feature modifies the block-builder StatefulSet which cannot be altered, so if it already exists it has to be deleted and re-applied again in order to be enabled.
    block_builder_concurrent_rollout_enabled: false,
    // Maximum number of unavailable replicas during a block-builder rollout when using block_builder_concurrent_rollout_enabled feature.
    // Computed from block-builder replicas by default, but can also be specified as percentage, for example "25%".
    block_builder_max_unavailable: $.tempo_block_builder_statefulset.spec.replicas,

    // disable tempo-query by default
    tempo_query: {
      enabled: false,
    },
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
    metrics_generator: {
      pvc_size: error 'Must specify a metrics-generator pvc size',
      pvc_storage_class: error 'Must specify a metrics-generator pvc storage class',
      ephemeral_storage_request_size: error 'Must specify a generator ephemeral_storage_request size',
      ephemeral_storage_limit_size: error 'Must specify a metrics generator ephemeral_storage_limit size',
      replicas: 0,
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
    block_builder: {
      replicas: 0,
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
    backend_scheduler: {
      replicas: 1,  // Only ever 1 backend-scheduler
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
    backend_worker: {
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
      tempoRetentionDuration: '336h',
      tempoSearchBackoffDuration: '5s',
      tempoReadBackoffDuration: '10s',
      tempoWriteBackoffDuration: '10s',
      tempoMetricsBackoffDuration: '0s',  // TraceQL Metrics checks disabled
      tempoLongWriteBackoffDuration: '50s',
    },
    ballast_size_mbs: '1024',
    port: 3200,
    http_api_prefix: '',
    gossip_ring_port: 7946,
    backend: error 'Must specify a backend',  // gcs|s3
    bucket: error 'Must specify a bucket',

    overrides_configmap_name: 'tempo-overrides',
    overrides+:: {
      super_user: {
        max_traces_per_user: 100000,
        ingestion_rate_limit_bytes: 200e5,  // ~20MB per sec
        ingestion_burst_size_bytes: 200e5,  // ~20MB
        max_bytes_per_trace: 300e5,  // ~30MB
      },
    },
  },
}
