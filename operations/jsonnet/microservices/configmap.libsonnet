{
  local configMap = $.core.v1.configMap,

  frigg_config:: {
    auth_enabled: false,
    server: {
      http_listen_port: $._config.port,
    },
    distributor: {
      receivers: $._config.distributor.receivers,
    },
    ingester: {
      trace_idle_period: '20s',
      traces_per_block: 100000,
      max_block_duration: '2h',
      flush_op_timeout: '1m',
      max_transfer_retries: 1,
      complete_block_timeout: '10m',
      lifecycler: {
        num_tokens: 512,
        heartbeat_period: '5s',
        join_after: '5s',
        ring: {
          heartbeat_timeout: '10m',
          replication_factor: 1,
          kvstore: {
            store: 'memberlist',
            memberlist: {
              abort_if_cluster_join_fails: false,
              bind_port: $._config.gossip_ring_port,
              join_members: ['gossip-ring.%s.svc.cluster.local:%d' % [$._config.namespace, $._config.gossip_ring_port]],
            },
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
  },

  frigg_compactor_config:: $.frigg_config
                           {
    compactor: {
      compaction: {
        chunkSizeBytes: 10485760,
        maxCompactionRange: '1h',
        blockRetention: '144h',
        compactedBlockRetention: '2m',
      },
    },
    storage_config+: {
      trace+: {
        maintenanceCycle: '10m',
        cache:: null,
      },
    },
  },

  frigg_configmap:
    configMap.new('frigg') +
    configMap.withData({
      'frigg.yaml': $.util.manifestYaml($.frigg_config),
    }) +
    configMap.withDataMixin({
      'overrides.yaml': |||
        overrides:
      |||,
    }),

  frigg_compactor_configmap:
    configMap.new('frigg-compactor') +
    configMap.withData({
      'frigg.yaml': $.util.manifestYaml($.frigg_compactor_config),
    }) +
    configMap.withDataMixin({
      'overrides.yaml': |||
        overrides:
      |||,
    }),

  frigg_query_configmap:
    configMap.new('frigg-query') +
    configMap.withData({
      'frigg-query.yaml': $.util.manifestYaml({
        backend: 'localhost:%d' % $._config.port,
      }),
    }),
}
