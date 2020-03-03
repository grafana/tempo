{
  local configMap = $.core.v1.configMap,

  frigg_configmap:
    configMap.new('frigg') +
    configMap.withData({
      'frigg.yaml': |||
        auth_enabled: false
        server:
          http_listen_port: 3100
        distributor:
          receivers:
            jaeger:
              protocols:
                grpc:
            opencensus:
        ingester:
          lifecycler:
            address: 127.0.0.1
            ring:
              kvstore:
                store: inmemory
              replication_factor: 1
            final_sleep: 0s
          trace_idle_period: 20s
          traces_per_block: 100000
          max_block_duration: 2h
          flush_op_timeout: 1m
          max_transfer_retries: 1
          complete_block_timeout: 10m
        compactor:
          compaction:
            chunkSizeBytes: 10485760 # 10 * 1024 * 1024
            maxCompactionRange: 2h
            blockRetention: 144h
            compactedBlockRetention: 2m
        storage_config:
          trace:
            maintenanceCycle: 5m
            backend: gcs
            wal:
              path: /var/frigg/wal
              bloom-filter-false-positive: .05
              index-downsample: 100
            gcs:
              bucket_name: ops-tools-frigg
              chunk_buffer_size: 10485760 # 1024 * 1024 * 10
            query_pool:
              max_workers: 50
              queue_depth: 10000
            cache:
              disk-path: /var/frigg/cache
              disk-max-mbs: 1024
              disk-prune-count: 100
              disk-clean-rate: 1m
        limits_config:
          enforce_metric_name: false
          reject_old_samples: true
          reject_old_samples_max_age: 168h
          per_tenant_override_config: /conf/overrides.yaml
      |||,
    }) +
    configMap.withDataMixin({
      'overrides.yaml': |||
        overrides:
      |||,
    }),

  frigg_query_configmap:
    configMap.new('frigg-query') +
    configMap.withData({
      'frigg-query.yaml': |||
        backend: "localhost:3100"
      |||,
    }),
}
