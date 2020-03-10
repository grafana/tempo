{
  local configMap = $.core.v1.configMap,

  tempo_config:: {
    auth_enabled: false,
    server: {
      http_listen_port: $._config.port
    },
    distributor: {
      receivers: $._config.receivers
    },
    ingester: {
      lifecycler: {
        address: "127.0.0.1",
        ring: {
          kvstore: {
            store: "inmemory"
          },
          replication_factor: 1
        },
        final_sleep: "0s"
      },
      trace_idle_period: "20s",
      traces_per_block: 100000,
      max_block_duration: "2h",
      flush_op_timeout: "1m",
      max_transfer_retries: 1,
      complete_block_timeout: "10m"
    },
    compactor: {
      compaction: {
        chunkSizeBytes: 10485760,
        maxCompactionRange: "2h",
        blockRetention: "144h",
        compactedBlockRetention: "2m"
      }
    },
    storage_config: {
      trace: {
        maintenanceCycle: "5m",
        backend: "local",
        wal: {
          path: "/var/tempo/wal",
          "bloom-filter-false-positive": 0.05,
          "index-downsample": 100
        },
        "local": {
          path: "/tmp/tempo/traces"
        },
        query_pool: {
          max_workers: 50,
          queue_depth: 10000
        },
        cache: {
          "disk-path": "/var/tempo/cache",
          "disk-max-mbs": 1024,
          "disk-prune-count": 100,
          "disk-clean-rate": "1m"
        }
      }
    },
    limits_config: {
      enforce_metric_name: false,
      reject_old_samples: true,
      reject_old_samples_max_age: "168h",
      per_tenant_override_config: "/conf/overrides.yaml"
    }
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

  tempo_query_configmap:
    configMap.new('tempo-query') +
    configMap.withData({
      'tempo-query.yaml': $.util.manifestYaml({
        backend: 'localhost:%d' % $._config.port
      })
    }),
}
