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
      trace_idle_period: '20s',
      traces_per_block: 200000,
      max_block_duration: '2h',
      flush_op_timeout: '10m',
      lifecycler: {
        num_tokens: 512,
        ring: {
          heartbeat_timeout: '5m',
          replication_factor: 2,
          kvstore: {
            store: 'memberlist',
          },
        },
      },
    },
    compactor: null,
    storage: {
      trace: {
        maintenance_cycle: '5m',
        backend: 'gcs',
        wal: {
          path: '/var/tempo/wal',
          bloom_filter_false_positive: 0.05,
          index_downsample: 100,
        },
        gcs: {
          bucket_name: $._config.gcs_bucket,
          chunk_buffer_size: 10485760,  // 1024 * 1024 * 10
        },
        pool: {
          max_workers: 50,
          queue_depth: 2000,
        },
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
        compaction_window: '4h',
        max_compaction_objects: 6000000,
        block_retention: '144h',
        compacted_block_retention: '2m',
      },
      ring: {
        kvstore: {
          store: 'memberlist',
        },
      },
    },
    storage+: {
      trace+: {
        maintenance_cycle: '10m',
      },
    },
  },

  tempo_querier_config:: $.tempo_config {
    server+: {
      log_level: 'debug',
    },
    storage+: {
      trace+: {
        pool+: {
          max_workers: 200,
        },
        memcached+: {
          timeout: '1s',
        },
      },
    },
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

  tempo_querier_configmap:
    configMap.new('tempo-querier') +
    configMap.withData({
      'tempo.yaml': $.util.manifestYaml($.tempo_querier_config),
    }),

  tempo_query_configmap:
    configMap.new('tempo-query') +
    configMap.withData({
      'tempo-query.yaml': $.util.manifestYaml({
        backend: 'localhost:%d' % $._config.port,
      }),
    }),
}
