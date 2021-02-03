{
  local configMap = $.core.v1.configMap,

  tempo_config:: {
    server: {
      http_listen_port: $._config.port,
    },
    distributor: {
      receivers: $._config.distributor.receivers,
    },
    ingester: {
      lifecycler: {
        ring: {
          replication_factor: 2,
        },
      },
    },
    compactor: null,
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
        pool: {
          queue_depth: 2000,
        },
        cache: 'memcached',
        memcached: {
          consistent_hash: true,
          timeout: '500ms',
          host: 'memcached',
          service: 'memcached-client',
        },
      },
    },
    overrides: {
      per_tenant_override_config: '/conf/overrides.yaml',
    },
    memberlist: {
      abort_if_cluster_join_fails: false,
      bind_port: $._config.gossip_ring_port,
      join_members: ['gossip-ring.%s.svc.cluster.local:%d' % [$._config.namespace, $._config.gossip_ring_port]],
    },
  },

  tempo_compactor_config:: $.tempo_config {
    compactor: {
      compaction: {
        chunk_size_bytes: 10485760,
        block_retention: '144h',
      },
      ring: {
        kvstore: {
          store: 'memberlist',
        },
      },
    },
    storage+: {
      trace+: {
        blocklist_poll: '10m',
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
        memcached+: {
          timeout: '1s',
        },
      },
    },
    querier+: {
      frontend_worker+: {
        frontend_address: 'query-frontend-discovery.%s.svc.cluster.local:9095' % [$._config.namespace],
      }
    }
  },

  tempo_configmap:
    configMap.new('tempo') +
    configMap.withData({
      'tempo.yaml': $.util.manifestYaml($.tempo_config),
    }) +
    configMap.withDataMixin({
      'overrides.yaml': $.util.manifestYaml({
        overrides: $._config.overrides,
      }),
    }),

  tempo_compactor_configmap:
    configMap.new('tempo-compactor') +
    configMap.withData({
      'tempo.yaml': $.util.manifestYaml($.tempo_compactor_config),
    }),

  tempo_query_frontend_config:: $.tempo_config{},

  tempo_querier_configmap:
    configMap.new('tempo-querier') +
    configMap.withData({
      'tempo.yaml': $.util.manifestYaml($.tempo_querier_config),
    }),

  tempo_query_frontend_configmap:
    configMap.new('tempo-query-frontend') +
    configMap.withData({
      'tempo.yaml': $.util.manifestYaml($.tempo_query_frontend_config),
    }),

  tempo_query_configmap:
    configMap.new('tempo-query') +
    configMap.withData({
      'tempo-query.yaml': $.util.manifestYaml({
        backend: 'localhost:%d' % $._config.port,
      }),
    }),
}
