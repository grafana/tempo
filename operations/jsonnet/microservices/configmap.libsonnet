{
  local configMap = $.core.v1.configMap,

  tempo_config:: {
    auth_enabled: false,
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
      max_transfer_retries: 1,
      complete_block_timeout: '30s',
      lifecycler: {
        num_tokens: 512,
        heartbeat_period: '5s',
        join_after: '5s',
        ring: {
          heartbeat_timeout: '10m',
          replication_factor: 3,
          kvstore: {
            store: 'memberlist',
          },
        },
      },
    },
    compactor: null,
    storage_config: {
      trace: {
        maintenanceCycle: '5m',
        backend: 'gcs',
        wal: {
          path: '/var/tempo/wal',
          'bloom-filter-false-positive': 0.05,
          'index-downsample': 100,
        },
        gcs: {
          bucket_name: $._config.gcs_bucket,
          chunk_buffer_size: 10485760,  // 1024 * 1024 * 10
        },
        query_pool: {
          max_workers: 50,
          queue_depth: 10000,
        },
        cache: {
          'disk-path': '/var/tempo/cache',
          'disk-max-mbs': 1024,
          'disk-prune-count': 100,
          'disk-clean-rate': '1m',
        },
      },
    },
    limits_config: {
      enforce_metric_name: false,
      reject_old_samples: true,
      reject_old_samples_max_age: '168h',
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
        chunkSizeBytes: 10485760,
        maxCompactionRange: '4h',
        blockRetention: '144h',
        compactedBlockRetention: '2m',
      },
      sharding_enabled: true,
      sharding_ring: {
        heartbeat_timeout: '10m',
        kvstore: {
          store: 'memberlist',
        },
      },
    },
    storage_config+: {
      trace+: {
        maintenanceCycle: '10m',
        cache:: null,
      },
    },
  },

  tempo_querier_config:: $.tempo_config {
    storage_config+: {
      trace+: {
        query_pool+: {
          max_workers: 500,
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
      'overrides.yaml': |||
        overrides:
      |||,
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
