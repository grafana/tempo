{
  local k = import 'ksonnet-util/kausal.libsonnet',
  local configMap = k.core.v1.configMap,

  tempo_config:: {
    http_api_prefix: $._config.http_api_prefix,

    server: {
      http_listen_port: $._config.port,
    },
    distributor: {},
    ingester: {
      lifecycler: {
        ring: {
          replication_factor: 3,
        },
      },
    },
    compactor: {},
    storage: {
      trace: {
        blocklist_poll: '0',
        backend: $._config.backend,
        wal: {
          path: '/var/tempo/wal',
        },
        gcs: {
          bucket_name: $._config.bucket,
          chunk_buffer_size: 10485760,  // 1024 * 1024 * 10
        },
        s3: {
          bucket: $._config.bucket,
        },
        azure: {
          container_name: $._config.bucket,
        },
        pool: {
          queue_depth: 2000,
        },
        cache: 'memcached',
        memcached: {
          consistent_hash: true,
          timeout: '200ms',
          host: 'memcached',
          service: 'memcached-client',
        },
      },
    },
    overrides: {
      per_tenant_override_config: '/overrides/overrides.yaml',
    },
    memberlist: {
      abort_if_cluster_join_fails: false,
      bind_port: $._config.gossip_ring_port,
      join_members: ['dns+gossip-ring.%s.svc.cluster.local.:%d' % [$._config.namespace, $._config.gossip_ring_port]],
    },
  },

  tempo_distributor_config:: $.tempo_config {
    distributor+: {
      receivers+: $._config.distributor.receivers,
    },
  },

  tempo_ingester_config:: $.tempo_config {},

  tempo_metrics_generator_config:: $.tempo_config {
    metrics_generator+: {
      storage+: {
        path: '/var/tempo/generator_wal',
      },
    },
  },

  tempo_compactor_config:: $.tempo_config {
    compactor+: {
      compaction+: {
        v2_in_buffer_bytes: 10485760,
        block_retention: '144h',
      },
      ring+: {
        kvstore+: {
          store: 'memberlist',
        },
      },
    },
    storage+: {
      trace+: {
        blocklist_poll: '5m',
      },
    },
  },

  tempo_querier_config:: $.tempo_config {
    server+: {
      log_level: 'debug',
    },
    storage+: {
      trace+: {
        blocklist_poll: '5m',
        pool+: {
          max_workers: 200,
        },
      },
    },
    querier+: {
      frontend_worker+: {
        frontend_address: 'query-frontend-discovery.%s.svc.cluster.local.:9095' % [$._config.namespace],
      },
    },
  },

  tempo_query_frontend_config:: $.tempo_config {},
  tempo_block_builder_config:: $.tempo_config {},
  tempo_backend_scheduler_config:: $.tempo_config {},

  // This will be the single configmap that stores `overrides.yaml`.
  overrides_config:
    configMap.new($._config.overrides_configmap_name) +
    configMap.withData({
      'overrides.yaml': k.util.manifestYaml({
        overrides: $._config.overrides,
      }),
    }),

  tempo_distributor_configmap:
    configMap.new('tempo-distributor') +
    configMap.withData({
      'tempo.yaml': k.util.manifestYaml($.tempo_distributor_config),
    }),

  tempo_ingester_configmap:
    configMap.new('tempo-ingester') +
    configMap.withData({
      'tempo.yaml': k.util.manifestYaml($.tempo_ingester_config),
    }),

  tempo_metrics_generator_configmap:
    configMap.new('tempo-metrics-generator') +
    configMap.withData({
      'tempo.yaml': $.util.manifestYaml($.tempo_metrics_generator_config),
    }),

  tempo_compactor_configmap:
    configMap.new('tempo-compactor') +
    configMap.withData({
      'tempo.yaml': k.util.manifestYaml($.tempo_compactor_config),
    }),

  tempo_querier_configmap:
    configMap.new('tempo-querier') +
    configMap.withData({
      'tempo.yaml': $.util.manifestYaml($.tempo_querier_config),
    }),

  tempo_block_builder_configmap:
    configMap.new('tempo-block-builder') +
    configMap.withData({
      'tempo.yaml': $.util.manifestYaml($.tempo_block_builder_config),
    }),

  tempo_query_frontend_configmap:
    configMap.new('tempo-query-frontend') +
    configMap.withData({
      'tempo.yaml': $.util.manifestYaml($.tempo_query_frontend_config),
    }),

  tempo_query_configmap: if $._config.tempo_query.enabled then
    configMap.new('tempo-query') +
    configMap.withData({
      'tempo-query.yaml': $.util.manifestYaml({
        backend: 'localhost:%d%s' % [$._config.port, $._config.http_api_prefix],
      }),
    }),
}
