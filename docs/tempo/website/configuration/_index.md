---
aliases:
- /docs/tempo/v1.2.1/configuration/
title: Configuration
weight: 200
---

# Configuration

This document explains the configuration options for Tempo as well as the details of what they impact. It includes:

  - [server](#server)
  - [distributor](#distributor)
  - [ingester](#ingester)
  - [query-frontend](#query-frontend)
  - [querier](#querier)
  - [compactor](#compactor)
  - [storage](#storage)
  - [memberlist](#memberlist)
  - [polling](#polling)
  - [overrides](#overrides)
  - [search](#search)

#### Use environment variables in the configuration

You can use environment variable references in the configuration file to set values that need to be configurable during deployment using `--config.expand-env` option.
To do this, use:

```
${VAR}
```

Where VAR is the name of the environment variable.

Each variable reference is replaced at startup by the value of the environment variable.
The replacement is case-sensitive and occurs before the YAML file is parsed.
References to undefined variables are replaced by empty strings unless you specify a default value or custom error text.

To specify a default value, use:

```
${VAR:-default_value}
```

where default_value is the value to use if the environment variable is undefined.

You can find more about other supported syntax [here](https://github.com/drone/envsubst/blob/master/readme.md)

## Server
Tempo uses the Weaveworks/common server. For more information on configuration options, see [here](https://github.com/weaveworks/common/blob/master/server/server.go#L54).

```yaml
# Optional. Setting to true enables multitenancy and requires X-Scope-OrgID header on all requests.
[multitenancy_enabled: <bool> | default = false]

# Optional. String prefix for all http api endpoints. Must include beginning slash.
[http_api_prefix: <string>]

server:
    # HTTP server listen host
    [http_listen_address: <string>]

    # HTTP server listen port
    [http_listen_port: <int> | default = 80]

    # gRPC server listen host
    [grpc_listen_address: <string>]

    # gRPC server listen port
    [grpc_listen_port: <int> | default = 9095]

    # Register instrumentation handlers (/metrics, etc.)
    [register_instrumentation: <boolean> | default = true]

    # Timeout for graceful shutdowns
    [graceful_shutdown_timeout: <duration> | default = 30s]

    # Read timeout for HTTP server
    [http_server_read_timeout: <duration> | default = 30s]

    # Write timeout for HTTP server
    [http_server_write_timeout: <duration> | default = 30s]

    # Idle timeout for HTTP server
    [http_server_idle_timeout: <duration> | default = 120s]

    # Max gRPC message size that can be received
    # This value may need to be increased if you have large traces
    [grpc_server_max_recv_msg_size: <int> | default = 4194304]

    # Max gRPC message size that can be sent
    # This value may need to be increased if you have large traces
    [grpc_server_max_send_msg_size: <int> | default = 4194304]
```

## Distributor
For more information on configuration options, see [here](https://github.com/grafana/tempo/blob/main/modules/distributor/config.go).

Distributors receive spans and forward them to the appropriate ingesters.

The following configuration enables all available receivers with their default configuration. For a production deployment, enable only the receivers you need.
Additional documentation and more advanced configuration options are available in [the receiver README](https://github.com/open-telemetry/opentelemetry-collector/blob/main/receiver/README.md).

```yaml
# Distributor config block
distributor:

    # receiver configuration for different protocols
    # config is passed down to opentelemetry receivers
    # for a production deployment you should only enable the receivers you need!
    receivers:
        otlp:
            protocols:
                grpc:
                http:
        jaeger:
            protocols:
                thrift_http:
                grpc:
                thrift_binary:
                thrift_compact:
        zipkin:
        opencensus:
        kafka:

    # Optional.
    # Enable to log every received trace id to help debug ingestion
    [log_received_traces: <bool>]

    # Optional.
    # disables write extension with inactive ingesters. Use this along with ingester.lifecycler.unregister_on_shutdown = true
    #  note that setting these two config values reduces tolerance to failures on rollout b/c there is always one guaranteed to be failing replica
    [extend_writes: <bool>]

    # Optional.
    # List of tags that will **not** be extracted from trace data for search lookups
    # This is a global config that will apply to all tenants
    [search_tags_deny_list: <list of string> | default = ]
```

## Ingester
For more information on configuration options, see [here](https://github.com/grafana/tempo/blob/main/modules/ingester/config.go).

The ingester is responsible for batching up traces and pushing them to [TempoDB](#storage).

```yaml
# Ingester configuration block
ingester:

    # Lifecycler is responsible for managing the lifecycle of entries in the ring.
    # For a complete list of config options check the lifecycler section under the ingester config at the following link -
    # https://cortexmetrics.io/docs/configuration/configuration-file/#ingester_config
    lifecycler:
        ring:
            # number of replicas of each span to make while pushing to the backend
            replication_factor: 3

    # amount of time a trace must be idle before flushing it to the wal.
    # (default: 10s)
    [trace_idle_period: <duration>]

    # how often to sweep all tenants and move traces from live -> wal -> completed blocks.
    # (default: 10s)
    [flush_check_period: <duration>]

    # maximum size of a block before cutting it
    # (default: 1073741824 = 1GB)
    [max_block_bytes: <int>]

    # maximum length of time before cutting a block
    # (default: 1h)
    [max_block_duration: <duration>]

    # duration to keep blocks in the ingester after they have been flushed
    # (default: 15m)
    [ complete_block_timeout: <duration>]
```

## Query-frontend
For more information on configuration options, see [here](https://github.com/grafana/tempo/blob/main/modules/frontend/config.go).

The Query Frontend is responsible for sharding incoming requests for faster processing in parallel (by the queriers).

```yaml
# Query Frontend configuration block
query_frontend:

    # number of times to retry a request sent to a querier
    # (default: 2)
    [max_retries: <int>]

    # number of shards to split the query into
    # (default: 20)
    [query_shards: <int>]

    # number of block queries that are tolerated to error before considering the entire query as failed
    # numbers greater than 0 make possible for a read to return partial results
    # partial results are indicated with HTTP status code 206
    # (default: 0)
    [tolerate_failed_blocks: <int>]
```

## Querier
For more information on configuration options, see [here](https://github.com/grafana/tempo/blob/main/modules/querier/config.go).

The Querier is responsible for querying the backends/cache for the traceID.

```yaml
# querier config block
querier:

    # Timeout for trace lookup requests
    [query_timeout: <duration> | default = 10s]

    # Timeout for search requests
    [search_query_timeout: <duration> | default = 30s]

    # Limit used for search requests if none is set by the caller
    [search_default_result_limit: <int> | default = 20]

    # The maximum allowed value of the limit parameter on search requests. If the search request limit parameter
    # exceeds the value configured here it will be set to the value configured here.
    # The default value of 0 disables this limit.
    [search_max_result_limit: <int> | default = 0]

    # config of the worker that connects to the query frontend
    frontend_worker:

        # the address of the query frontend to connect to, and process queries
        # Example: "frontend_address: query-frontend-discovery.default.svc.cluster.local:9095"
        [frontend_address: <string>]
```

It also queries compacted blocks that fall within the (2 * BlocklistPoll) range where the value of Blocklist poll duration
is defined in the storage section below.

## Compactor
For more information on configuration options, see [here](https://github.com/grafana/tempo/blob/main/modules/compactor/config.go).

Compactors stream blocks from the storage backend, combine them and write them back.  Values shown below are the defaults.

```yaml
compactor:

    ring:

        kvstore:

            # in a high volume environment multiple compactors need to work together to keep up with incoming blocks.
            # this tells the compactors to use a ring stored in memberlist to coordinate.
            # Example: "store: memberlist"
            [store: <string>]

    compaction:

        # Optional. Duration to keep blocks.  Default is 14 days (336h).
        [block_retention: <duration>]

        # Optional. Duration to keep blocks that have been compacted elsewhere. Default is 1h.
        [compacted_block_retention: <duration>]

        # Optional. Blocks in this time window will be compacted together. Default is 1h.
        [compaction_window: <duration>]

        # Optional. Amount of data to buffer from input blocks. Default is 5 MiB.
        [chunk_size_bytes: <int>]

        # Optional. Flush data to backend when buffer is this large. Default is 30 MB.
        [flush_size_bytes: <int>]

        # Optional. Maximum number of traces in a compacted block. Default is 6 million.
        # WARNING: Deprecated. Use max_block_bytes instead.
        [max_compaction_objects: <int>]

        # Optional. Maximum size of a compacted block in bytes.  Default is 100 GB.
        [max_block_bytes: <int>]

        # Optional. Number of tenants to process in parallel during retention. Default is 10.
        [retention_concurrency: <int>]

        # Optional. Number of traces to buffer in memory during compaction. Increasing may improve performance but will also increase memory usage. Default is 1000.
        [iterator_buffer_size: <int>]
```

## Storage
For more information on configuration options, see [here](https://github.com/grafana/tempo/blob/main/tempodb/config.go).

The storage block is used to configure TempoDB. It supports S3, GCS, Azure, local file system, and optionally can use Memcached or Redis for increased query performance.

The following example shows common options.  For further platform-specific information refer to the following:
* [GCS](gcs/)
* [S3](s3/)

```yaml
# Storage configuration for traces
storage:

    trace:

        # The storage backend to use
        # Should be one of "gcs", "s3", "azure" or "local"
        # CLI flag -storage.trace.backend
        [backend: <string>]

        # GCS configuration. Will be used only if value of backend is "gcs"
        # Check the GCS doc within this folder for information on GCS specific permissions.
        gcs:

            # Bucket name in gcs
            # Tempo requires a dedicated bucket since it maintains a top-level object structure and does not support
            # a custom prefix to nest within a shared bucket.
            # Example: "bucket_name: tempo"
            [bucket_name: <string>]

            # Buffer size for reads. Default is 10MB
            # Example: "chunk_buffer_size: 5_000_000"
            [chunk_buffer_size: <int>]

            # Optional
            # Api endpoint override
            # Example: "endpoint: https://storage.googleapis.com/storage/v1/"
            [endpoint: <string>]

            # Optional. Default is false.
            # Example: "insecure: true"
            # Set to true to enable authentication and certificate checks on gcs requests
            [insecure: <bool>]

            # Optional. Default is 0 (disabled)
            # Example: "hedge_requests_at: 500ms"
            # If set to a non-zero value a second request will be issued at the provided duration. Recommended to
            # be set to p99 of GCS requests to reduce long tail latency. This setting is most impactful when
            # used with queriers and has minimal to no impact on other pieces.
            [hedge_requests_at: <duration>]

        # S3 configuration. Will be used only if value of backend is "s3"
        # Check the S3 doc within this folder for information on s3 specific permissions.
        s3:

            # Bucket name in s3
            # Tempo requires a dedicated bucket since it maintains a top-level object structure and does not support
            # a custom prefix to nest within a shared bucket.
            [bucket: <string>]

            # api endpoint to connect to. use AWS S3 or any S3 compatible object storage endpoint.
            # Example: "endpoint: s3.dualstack.us-east-2.amazonaws.com"
            [endpoint: <string>]

            # optional.
            # By default the region is inferred from the endpoint,
            # but is required for some S3-compatible storage engines.
            # Example: "region: us-east-2"
            [region: <string>]

            # optional.
            # access key when using static credentials.
            [access_key: <string>]

            # optional.
            # secret key when using static credentials.
            [secret_key: <string>]

            # optional.
            # enable if endpoint is http
            [insecure: <bool>]

            # optional.
            # enable to use path-style requests.
            [forcepathstyle: <bool>]

            # Optional. Default is 0 (disabled)
            # Example: "hedge_requests_at: 500ms"
            # If set to a non-zero value a second request will be issued at the provided duration. Recommended to
            # be set to p99 of S3 requests to reduce long tail latency.  This setting is most impactful when
            # used with queriers and has minimal to no impact on other pieces.
            [hedge_requests_at: <duration>]

        # azure configuration. Will be used only if value of backend is "azure"
        # EXPERIMENTAL
        azure:

            # store traces in this container.
            # Tempo requires a dedicated bucket since it maintains a top-level object structure and does not support
            # a custom prefix to nest within a shared bucket.
            [container-name: <string>]

            # optional.
            # Azure endpoint to use, defaults to Azure global(core.windows.net) for other
            # regions this needs to be changed e.g Azure China(blob.core.chinacloudapi.cn),
            # Azure German(blob.core.cloudapi.de), Azure US Government(blob.core.usgovcloudapi.net).
            [endpoint-suffix: <string>]

            # Name of the azure storage account
            [storage-account-name: <string>]

            # optional.
            # access key when using access key credentials.
            [storage-account-key: <string>]

            # Optional. Default is 0 (disabled)
            # Example: "hedge-requests-at: 500ms"
            # If set to a non-zero value a second request will be issued at the provided duration. Recommended to
            # be set to p99 of Axure Blog Storage requests to reduce long tail latency.  This setting is most impactful when
            # used with queriers and has minimal to no impact on other pieces.
            [hedge-requests-at: <duration>]

        # How often to repoll the backend for new blocks. Default is 5m
        [blocklist_poll: <duration>]

        # Number of blocks to process in parallel during polling. Default is 50.
        [blocklist_poll_concurrency: <int>]

        # By default components will pull the blocklist from the tenant index. If that fails the component can
        # fallback to scanning the entire bucket. Set to false to disable this behavior. Default is true.
        [blocklist_poll_fallback: <bool>]

        # Maximum number of compactors that should build the tenant index. All other components will download
        # the index.  Default 2.
        [blocklist_poll_tenant_index_builders: <int>]

        # The oldest allowable tenant index. If an index is pulled that is older than this duration,
        # the polling will consider this an error. Note that `blocklist_poll_fallback` applies here.
        # If fallback is true and a tenant index exceeds this duration, it will fall back to listing
        # the bucket contents.
        # Default 0 (disabled).
        [blocklist_poll_stale_tenant_index: <duration>]

        # Cache type to use. Should be one of "redis", "memcached"
        # Example: "cache: memcached"
        [cache: <string>]

        # Minimum compaction level of block to qualify for bloom filter caching. Default is 0 (disabled), meaning
        # that compaction level is not used to determine if the bloom filter should be cached.
        # Example: "cache_min_compaction_level: 2"
        [cache_min_compaction_level: <int>]

        # Max block age to qualify for bloom filter caching. Default is 0 (disabled), meaning that block age is not
        # used to determine if the bloom filter should be cached.
        # Example: "cache_max_block_age: 48h"
        [cache_max_block_age: <duration>]

        # Cortex Background cache configuration. Requires having a cache configured.
        background_cache:

            # at what concurrency to write back to cache. Default is 10.
            [writeback_goroutines: <int>]

            # how many key batches to buffer for background write-back. Default is 10000.
            [writeback_buffer: <int>]

        # Memcached caching configuration block
        memcached:

            # hostname for memcached service to use. If empty and if addresses is unset, no memcached will be used.
            # Example: "host: memcached"
            [host: <string>]

            # Optional
            # SRV service used to discover memcache servers. (default: memcached)
            # Example: "service: memcached-client"
            [service: <string>]

            # Optional
            # comma separated addresses list in DNS Service Discovery format. Refer - https://cortexmetrics.io/docs/configuration/arguments/#dns-service-discovery.
            # (default: "")
            # Example: "addresses: memcached"
            [addresses: <comma separated strings>]

            # Optional
            # Maximum time to wait before giving up on memcached requests.
            # (default: 100ms)
            [timeout: <duration>]

            # Optional
            # Maximum number of idle connections in pool.
            # (default: 16)
            [max_idle_conns: <int>]

            # Optional
            # period with which to poll DNS for memcache servers.
            # (default: 1m)
            [update_interval: <duration>]

            # Optional
            # use consistent hashing to distribute keys to memcache servers.
            # (default: true)
            [consistent_hash: <bool>]

            # Optional
            # trip circuit-breaker after this number of consecutive dial failures.
            # (default: 10)
            [circuit_breaker_consecutive_failures: 10]

            # Optional
            # duration circuit-breaker remains open after tripping.
            # (default: 10s)
            [circuit_breaker_timeout: 10s]

            # Optional
            # reset circuit-breaker counts after this long.
            # (default: 10s)
            [circuit_breaker_interval: 10s]

        # Redis configuration block
        # EXPERIMENTAL
        redis:

            # redis endpoint to use when caching.
            [endpoint: <string>]

            # optional.
            # maximum time to wait before giving up on redis requests. (default 100ms)
            [timeout: 500ms]

            # optional.
            # redis Sentinel master name. (default "")
            # Example: "master-name: redis-master"
            [master-name: <string>]

            # optional.
            # database index. (default 0)
            [db: <int>]

            # optional.
            # how long keys stay in the redis. (default 0)
            [expiration: <duration>]

            # optional.
            # enable connecting to redis with TLS. (default false)
            [tls-enabled: <bool>]

            # optional.
            # skip validating server certificate. (default false)
            [tls-insecure-skip-verify: <bool>]

            # optional.
            # maximum number of connections in the pool. (default 0)
            [pool-size: <int>]

            # optional.
            # password to use when connecting to redis. (default "")
            [password: <string>]

            # optional.
            # close connections after remaining idle for this duration. (default 0s)
            {idle-timeout: <duration>}

            # optional.
            # close connections older than this duration. (default 0s)
            [max-connection-age: <duration>]

        # the worker pool is used primarily when finding traces by id, but is also used by other
        pool:

            # total number of workers pulling jobs from the queue (default: 30)
            [max_workers: <int>]

            # length of job queue. imporatant for querier as it queues a job for every block it has to search
            # (default: 10000)
            [queue_depth: <int>]

        # Configuration block for the Write Ahead Log (WAL)
        wal:

            # where to store the head blocks while they are being appended to
            # Example: "wal: /var/tempo/wal"
            [path: <string>]

            # wal encoding/compression.
            # options: none, gzip, lz4-64k, lz4-256k, lz4-1M, lz4, snappy, zstd, s2
            # (default: snappy)
            [encoding: <string>]

            # search data encoding/compression. same options as wal encoding.
            # (default: none)
            [search_encoding: <string>]

        # block configuration
        block:

            # bloom filter false positive rate.  lower values create larger filters but fewer false positives
            # (default: .01)
            [bloom_filter_false_positive: <float>]

            # maximum size of each bloom filter shard
            # (default: 100 KiB)
            [bloom_filter_shard_size_bytes: <int>]

            # number of bytes per index record
            # (default: 1MiB)
            [index_downsample_bytes: <uint64>]

            # block encoding/compression.  options: none, gzip, lz4-64k, lz4-256k, lz4-1M, lz4, snappy, zstd, s2
            [encoding: <string>]

            # search data encoding/compression. same options as block encoding.
            # (default: snappy)
            [search_encoding: <string>]

            # number of bytes per search page
            # (default: 1MiB)
            [search_page_size_bytes: <int>]

```

## Memberlist
[Memberlist](https://github.com/hashicorp/memberlist) is the default mechanism for all of the Tempo pieces to coordinate with each other.

```yaml
memberlist:
    # Name of the node in memberlist cluster. Defaults to hostname.
    [node_name: <string> | default = ""]

    # Add random suffix to the node name.
    [randomize_node_name: <boolean> | default = true]

    # The timeout for establishing a connection with a remote node, and for
    # read/write operations.
    [stream_timeout: <duration> | default = 10s]

    # Multiplication factor used when sending out messages (factor * log(N+1)).
    [retransmit_factor: <int> | default = 2]

    # How often to use pull/push sync.
    [pull_push_interval: <duration> | default = 30s]

    # How often to gossip.
    [gossip_interval: <duration> | default = 1s]

    # How many nodes to gossip to.
    [gossip_nodes: <int> | default = 2]

    # How long to keep gossiping to dead nodes, to give them chance to refute their
    # death.
    [gossip_to_dead_nodes_time: <duration> | default = 30s]

    # How soon can dead node's name be reclaimed with new address. Defaults to 0,
    # which is disabled.
    [dead_node_reclaim_time: <duration> | default = 0s]

    # Other cluster members to join. Can be specified multiple times. It can be an
    # IP, hostname or an entry specified in the DNS Service Discovery format (see
    # https://cortexmetrics.io/docs/configuration/arguments/#dns-service-discovery
    # for more details).
    # A "Headless" Cluster IP service in Kubernetes.
    # Example:
    #   - gossip-ring.tracing.svc.cluster.local:7946
    [join_members: <list of string> | default = ]

    # Min backoff duration to join other cluster members.
    [min_join_backoff: <duration> | default = 1s]

    # Max backoff duration to join other cluster members.
    [max_join_backoff: <duration> | default = 1m]

    # Max number of retries to join other cluster members.
    [max_join_retries: <int> | default = 10]

    # If this node fails to join memberlist cluster, abort.
    [abort_if_cluster_join_fails: <boolean> | default = true]

    # If not 0, how often to rejoin the cluster. Occasional rejoin can help to fix
    # the cluster split issue, and is harmless otherwise. For example when using
    # only few components as a seed nodes (via -memberlist.join), then it's
    # recommended to use rejoin. If -memberlist.join points to dynamic service that
    # resolves to all gossiping nodes (eg. Kubernetes headless service), then rejoin
    # is not needed.
    [rejoin_interval: <duration> | default = 0s]

    # How long to keep LEFT ingesters in the ring.
    [left_ingesters_timeout: <duration> | default = 5m]

    # Timeout for leaving memberlist cluster.
    [leave_timeout: <duration> | default = 5s]

    # IP address to listen on for gossip messages. Multiple addresses may be
    # specified. Defaults to 0.0.0.0
    [bind_addr: <list of string> | default = ]

    # Port to listen on for gossip messages.
    [bind_port: <int> | default = 7946]

    # Timeout used when connecting to other nodes to send packet.
    [packet_dial_timeout: <duration> | default = 5s]

    # Timeout for writing 'packet' data.
    [packet_write_timeout: <duration> | default = 5s]

```

## Overrides
Tempo provides a overrides module for user to set global or per-tenant override settings.
**Currently only ingestion limits can be overridden.**

### Ingestion limits
The default limits in Tempo may not be sufficient in high volume tracing environments. Errors including `RATE_LIMITED`/`TRACE_TOO_LARGE`/`LIVE_TRACES_EXCEEDED` will occur when these limits are exceeded.

#### Standard overrides
You can create an `overrides` section to configure new ingestion limits that applies to all tenants of the cluster.
A snippet of a config.yaml file showing how the overrides section is [here](https://github.com/grafana/tempo/blob/a000a0d461221f439f585e7ed55575e7f51a0acd/integration/bench/config.yaml#L39-L40).

```yaml
# Overrides configuration block
overrides:

    # Global ingestion limits configurations

    # Burst size (bytes) used in ingestion.
    # (Default: `20,000,000` ~20MB )
    # Results in errors like
    #   RATE_LIMITED: ingestion rate limit (15000000 bytes) exceeded while adding 10 bytes
    [ingestion_burst_size_bytes: <int>]

    # Per-user ingestion rate limit (bytes) used in ingestion.
    # (Default: `15,000,000` ~15MB)
    # Results in errors like
    #   RATE_LIMITED: ingestion rate limit (15000000 bytes) exceeded while
    [ingestion_rate_limit_bytes: <int>]

    # Maximum size of a single trace in bytes.  `0` to disable.
    # (Default: `5,000,000` ~5MB)
    # Results in errors like
    #    TRACE_TOO_LARGE: max size of trace (5000000) exceeded while adding 387 bytes
    [max_bytes_per_trace: <int>]

    # Maximum number of active traces per user, per ingester. `0` to disable.
    # (Default: `10,000`)
    # Results in errors like
    #    LIVE_TRACES_EXCEEDED: max live traces per tenant exceeded: per-user traces limit (local: 10000 global: 0 actual local: 1) exceeded
    [max_traces_per_user: <int> ]


    # Tenant-specific overrides

    # tenant-specific overrides settings config file
    [per_tenant_override_config: /conf/overrides.yaml]

    # Ingestion strategy, default is `local`.
    [ingestion_rate_strategy: <global|local>]
```


#### Tenant-specific overrides

You can set tenant-specific overrides settings in a separate file and point `per_tenant_override_config` to it. This overrides file is dynamically loaded.  It can be changed at runtime and will be reloaded by Tempo without restarting the application.
```yaml
# /conf/tempo.yaml
# Overrides configuration block
overrides:
   per_tenant_override_config: /conf/overrides.yaml

---
# /conf/overrides.yaml
# Tenant-specific overrides configuration
overrides:

    "<tenant id>":
        [ingestion_burst_size_bytes: <int>]
        [ingestion_rate_limit_bytes: <int>]
        [max_bytes_per_trace: <int>]
        [max_traces_per_user: <int>]

    # A "wildcard" override can be used that will apply to all tenants if a match is not found otherwise.
    "*":
        [ingestion_burst_size_bytes: <int>]
        [ingestion_rate_limit_bytes: <int>]
        [max_bytes_per_trace: <int>]
        [max_traces_per_user: <int>]
```

#### Override strategies

The trace limits specified by the various parameters are, by default, applied as per-distributor limits. For example, a `max_traces_per_user` setting of 10000 means that each distributor within the cluster has a limit of 10000 traces per user. This is known as a `local` strategy in that the specified trace limits are local to each distributor.

A setting that applies at a local level is quite helpful in ensuring that each distributor independently can process traces up to the limit without affecting the tracing limits on other distributors.

However, as a cluster grows quite large, this can lead to quite a large quantity of traces. An alternative strategy may be to set a `global` trace limit that establishes a total budget of all traces across all distributors in the cluster. The global limit is averaged across all distributors by using the distributor ring.
```yaml
# /conf/tempo.yaml
overrides:
    # Ingestion strategy, default is `local`.
    ingestion_rate_strategy: <global|local>
```

## Search

Tempo native search can be enabled by the following top-level setting.  In microservices mode, it must be set for the distributors and queriers.

```yaml
search_enabled: true
```

Additional search-related settings are available in the [distributor](#distributor) and [ingester](#ingester) sections.
