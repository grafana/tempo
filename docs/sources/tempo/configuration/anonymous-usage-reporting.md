---
title: Anonymous usage reporting
description: Learn about anonymous usage statistics reporting in Tempo.
weight: 950
---

# Anonymous usage reporting

By default, Tempo reports anonymous usage data about itself to Grafana Labs.
This data is used to understand which features are commonly enabled, as well as which deployment modes, replication factors, and compression levels are most popular, etc.

By providing information on how people use Tempo, usage reporting helps the Tempo team decide where to focus their development and documentation efforts.

The following configuration values are used:

- Receivers enabled
- Frontend concurrency and version
- Storage cache, backend, WAL and block encodings
- Ring replication factor, and `kvstore`
- Features toggles enabled

No private information is collected, and all reports are completely anonymous.

## Configure anonymous usage reporting

Reporting is controlled by the `usage_report` configuration option and can be disabled.
For instructions, refer to [the Configuration documentation](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#usage-report).

## View usage stats report

Tempo provides a usage stats report that you can view in your browser.

To view the report, go to the following URL on the Tempo instance: `http://localhost:3200/status/usage-stats`

An example report output looks like this:

```json
{
  "clusterID": "",
  "createdAt": "0001-01-01T00:00:00Z",
  "interval": "0001-01-01T00:00:00Z",
  "intervalPeriod": 14400,
  "target": "all",
  "version": {
    "version": "v2.8.0",
    "revision": "31e2dddb5",
    "branch": "main",
    "buildUser": "",
    "buildDate": "",
    "goVersion": "go1.24.3"
  },
  "os": "linux",
  "arch": "arm64",
  "edition": "oss",
  "metrics": {
    "ring_kv_store": "inmemory",
    "memstats": {
      "heap_inuse": 14106624,
      "stack_inuse": 2490368,
      "pause_total_ns": 872084,
      "num_gc": 3,
      "gc_cpu_fraction": 0.08786506719453682,
      "heap_alloc": 11640400,
      "alloc": 11640400,
      "total_alloc": 16491760,
      "sys": 27874568
    },
    "num_cpu": 8,
    "feature_enabled_multitenancy": 0,
    "receiver_enabled_jaeger": 0,
    "storage_block_encoding": "zstd",
    "storage_block_search_encoding": "snappy",
    "storage_cache": "",
    "receiver_enabled_opencensus": 0,
    "feature_enabled_auth_stats": 0,
    "frontend_version": "v1",
    "storage_backend": "local",
    "receiver_enabled_otlp": 0,
    "ring_replication_factor": 1,
    "storage_wal_encoding": "snappy",
    "storage_wal_search_encoding": "none",
    "num_goroutine": 813,
    "cache_memcached": 1,
    "cache_redis": 0,
    "distributor_bytes_received": {
      "total": 0,
      "rate": 0
    },
    "distributor_spans_received": {
      "total": 0,
      "rate": 0
    },
    "receiver_enabled_kafka": 0,
    "receiver_enabled_zipkin": 0
  }
}
```

## Which information is collected?

Tempo collects and reports the following information to Grafana Labs. 
The report from your Tempo instance may vary from the provided example. 
Each field provides insight into the Tempo instance, its environment, and configuration. The fields are grouped by their purpose.

This information helps Grafana Labs understand how Tempo is used, which features are enabled, and the typical deployment environments, without collecting any private or user-identifying data.

{{< admonition type="note">}}
Tempo maintainers commit to keeping the list of tracked information updated over time, and reporting any change both via the CHANGELOG and the release notes.
{{< /admonition>}}

### Instance identification

- **`clusterID`**: A unique, randomly generated identifier for the Tempo cluster. This value helps Grafana Labs distinguish between different deployments.
- **`createdAt`**: The timestamp when anonymous usage reporting was first enabled and the cluster ID was created.
- **`interval`**: The timestamp marking the start of the current reporting interval.
- **`intervalPeriod`**: The length of the reporting interval, in seconds.

### Deployment and version information

- **`target`**: The deployment mode or target for the Tempo instance, such as `all` for monolithic mode.
- **`version`**: An object containing detailed version information:
  - **`version`**: The Tempo version, for example `v2.8.0`.
  - **`revision`**: The Git commit hash or revision used to build the binary.
  - **`branch`**: The Git branch used for the build.
  - **`buildUser`**: The user who built the binary.
  - **`buildDate`**: The date and time when the binary was built.
  - **`goVersion`**: The Go language version used for the build.

### Environment details

- **`os`**: The operating system the Tempo instance is running on, such as `linux`.
- **`arch`**: The system architecture, such as `arm64`.
- **`edition`**: The edition of Tempo, such as `oss` for open source.

### Metrics and configuration

- **`metrics`**: An object containing runtime metrics and configuration:
  - **`ring_kv_store`**: The key-value store used for the ring, for example `inmemory`.
  - **`memstats`**: Memory usage statistics, including heap and stack usage, garbage collection metrics, and total allocations.
  - **`num_cpu`**: The number of logical CPU cores available.
  - **`feature_enabled_multitenancy`**: Indicates if multitenancy is enabled (`1`) or not (`0`).
  - **`receiver_enabled_jaeger`**, **`receiver_enabled_opencensus`**, **`receiver_enabled_otlp`**, **`receiver_enabled_kafka`**, **`receiver_enabled_zipkin`**: Flags indicating if each trace receiver is enabled (`1`) or not (`0`).
  - **`storage_block_encoding`**, **`storage_block_search_encoding`**, **`storage_wal_encoding`**, **`storage_wal_search_encoding`**: The encoding or compression algorithms used for storage blocks and write-ahead logs.
  - **`storage_cache`**: The cache backend used for storage, if any.
  - **`feature_enabled_auth_stats`**: Indicates if authentication statistics are enabled.
  - **`frontend_version`**: The version of the frontend component.
  - **`storage_backend`**: The storage backend in use, such as `local`.
  - **`ring_replication_factor`**: The replication factor for the ring.
  - **`num_goroutine`**: The number of active Go routines.
  - **`cache_memcached`**, **`cache_redis`**: Flags indicating if Memcached or Redis caching is enabled.
  - **`distributor_bytes_received`**, **`distributor_spans_received`**: Objects showing the total and rate of bytes and spans received by the distributor.
