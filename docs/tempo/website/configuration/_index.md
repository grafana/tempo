---
title: Configuration
weight: 200
---

# Configuration

This section explains the configuration options for Tempo as well as the details of what they impact. It includes:

  - [Authentication/Server](#authenticationserver)
  - [Distributor](#distributor)
  - [Ingester](#ingester)
  - [Query Frontend](#queryfrontend)
  - [Querier](#querier)
  - [Compactor](#compactor)
  - [Storage](#storage)
  - [Memberlist](#memberlist)
  - [Compression](#compression)

## Authentication/Server
Tempo uses the Weaveworks/common server.  See [here](https://github.com/weaveworks/common/blob/main/server/server.go#L45) for all configuration options.

```
auth_enabled: true            # Optional. Require X-Scope-OrgID. By default, it's set to false (disabled).
server:
  http_listen_port: 3100
```

## Distributor
See [here](https://github.com/grafana/tempo/blob/main/modules/distributor/config.go) for all configuration options.

Distributors are responsible for receiving spans and forwarding them to the appropriate ingesters.  The below configuration
exposes the otlp receiver on port 0.0.0.0:5680.  [This configuration](https://github.com/grafana/tempo/blob/main/example/docker-compose/etc/tempo-s3-minio.yaml) shows how to
configure all available receiver options.

```
distributor:
    receivers:
        otlp:
            protocols:
                grpc:
                    endpoint: 0.0.0.0:55680
```

## Ingester
See [here](https://github.com/grafana/tempo/blob/main/modules/ingester/config.go) for all configuration options.

The ingester is responsible for batching up traces and pushing them to [TempoDB](#storage).

```
ingester:
    lifecycler:
        ring:
            replication_factor: 3   # number of replicas of each span to make while pushing to the backend
    trace_idle_period: 20s          # amount of time before considering a trace complete and flushing it to a block
    max_block_bytes: 1_000_000_000  # maximum size of a block before cutting it
    max_block_duration: 1h          # maximum length of time before cutting a block
```

## Query Frontend
See [here](https://github.com/grafana/tempo/blob/main/modules/frontend/config.go) for all configuration options.

The Query Frontend is responsible for sharding incoming requests for faster processing in parallel (by the queriers).

```
query_frontend:
    query_shards: 10    # number of shards to split the query into
```

## Querier
See [here](https://github.com/grafana/tempo/blob/main/modules/querier/config.go) for all configuration options.

The Querier is responsible for querying the backends/cache for the traceID.

```
querier:
    frontend_worker:
        frontend_address: query-frontend-discovery.default.svc.cluster.local:9095   # the address of the query frontend to connect to, and process queries
```

It also queries compacted blocks that fall within the (2 * BlocklistPoll) range where the value of Blocklist poll duration
is defined in the storage section below.

## Compactor
See [here](https://github.com/grafana/tempo/blob/main/modules/compactor/config.go) for all configuration options.

Compactors stream blocks from the storage backend, combine them and write them back.  Values shown below are the defaults.

```
compactor:
    compaction:
        block_retention: 336h               # Optional. Duration to keep blocks.  Default is 14 days (336h).
        compacted_block_retention: 1h       # Optional. Duration to keep blocks that have been compacted elsewhere
        compaction_window: 4h               # Optional. Blocks in this time window will be compacted together
        chunk_size_bytes: 10485760          # Optional. Amount of data to buffer from input blocks. Default is 10 MiB
        flush_size_bytes: 31457280          # Optional. Flush data to backend when buffer is this large. Default is 30 MiB
        max_compaction_objects: 6000000     # Optional. Maximum number of traces in a compacted block. Default is 6 million. Deprecated.
        max_block_bytes: 107374182400       # Optional. Maximum size of a compacted block in bytes.  Default is 100 GiB
        retention_concurrency: 10           # Optional. Number of tenants to process in parallel during retention. Default is 10.
    ring:
        kvstore:
            store: memberlist       # in a high volume environment multiple compactors need to work together to keep up with incoming blocks.
                                    # this tells the compactors to use a ring stored in memberlist to coordinate.
```

## Storage
See [here](https://github.com/grafana/tempo/blob/main/tempodb/config.go) for all configuration options.

The storage block is used to configure TempoDB. It supports S3, GCS, Azure, local file system, and optionally can use Memcached or Redis for increased query performance.  

The following example shows common options.  For platform-specific options refer to the following:
* [Azure](azure/)
* [GCS](gcs/)
* [S3](s3/)
* [Memcached](memcached/)
* [Redis](redis/) (experimental)

```
storage:
    trace:
        backend: gcs                             # store traces in gcs
        gcs:
            bucket_name: ops-tools-tracing-ops   # store traces in this bucket
        blocklist_poll: 5m                       # how often to repoll the backend for new blocks
        blocklist_poll_concurrency: 50           # optional. Number of blocks to process in parallel during polling. Default is 50.
        cache: memcached                         # optional. Cache configuration
        background_cache:                        # optional. Background cache configuration. Requires having a cache configured.
            writeback_goroutines: 10             # at what concurrency to write back to cache. Default is 10.
            writeback_buffer: 10000              # how many key batches to buffer for background write-back. Default is 10000.
        memcached:                               # optional. Memcached configuration
            consistent_hash: true
            host: memcached
            service: memcached-client
            timeout: 500ms
        redis:                                   # optional. Redis configuration 
            endpoint: redis
            timeout: 500ms
        pool:                                    # the worker pool is used primarily when finding traces by id, but is also used by other
            max_workers: 50                      # total number of workers pulling jobs from the queue
            queue_depth: 2000                    # length of job queue
        wal:
            path: /var/tempo/wal                 # where to store the head blocks while they are being appended to
            encoding: none                       # (experimental) wal encoding/compression.  options: none, gzip, lz4-64k, lz4-256k, lz4-1M, lz4, snappy, zstd   
        block:
            bloom_filter_false_positive: .05     # bloom filter false positive rate.  lower values create larger filters but fewer false positives
            index_downsample_bytes: 1_000_000    # number of bytes per index record 
            encoding: zstd                       # block encoding/compression.  options: none, gzip, lz4-64k, lz4-256k, lz4-1M, lz4, snappy, zstd
```

## Memberlist
[Memberlist](https://github.com/hashicorp/memberlist) is the default mechanism for all of the Tempo pieces to coordinate with each other.

```
memberlist:
    bind_port: 7946
    join_members:
      - gossip-ring.tracing-ops.svc.cluster.local:7946  # A DNS entry that lists all tempo components.  A "Headless" Cluster IP service in Kubernetes
```
