---
title: Configuration
draft: true
---

### Authentication

```
auth_enabled: false

server:
  http_listen_port: 3100
```

### Distributor
```
distributor:
  receivers:
    jaeger:
      protocols:
        thrift_http:
```

### Ingester

```
ingester:
  lifecycler:
    address: 127.0.0.1
    ring:
      kvstore:
        store: inmemory
      replication_factor: 1
    final_sleep: 0s
  trace_idle_period: 10s  # how long to wait until we call a trace complete
  traces_per_block: 100
  max_block_duration: 5m
```

### Compactor
```

compactor:
  compaction:
    chunk_size_bytes: 1048576 # 1024 * 1024
    compaction_window: 1h
    max_compaction_objects: 1000000
    block_retention: 1h
    compacted_block_retention: 10m
```

### TempoDB config

```
storage:
  trace:
    maintenance_cycle: 30s
    backend: s3
    wal:
      path: /tmp/tempo/wal
      bloom_filter_false_positive: .05
      index_downsample: 10
    s3:
      bucket: tempo
      endpoint: minio:9000
      access_key: tempo
      secret_key: supersecret
      insecure: true
    pool:
      max_workers: 100
      queue_depth: 10000
    disk_cache:
      disk_path: /tmp/tempo/cache
      disk_max_mbs: 5
      disk_prune_count: 10
      disk_clean_rate: 30s
```