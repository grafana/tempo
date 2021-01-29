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

## Authentication/Server
Tempo uses the Weaveworks/common server.  See [here](https://github.com/weaveworks/common/blob/master/server/server.go#L45) for all configuration options.

```
auth_enabled: false            # do not require X-Scope-OrgID
server:
  http_listen_port: 3100
```

## Distributor
See [here](https://github.com/grafana/tempo/blob/master/modules/distributor/config.go) for all configuration options.

Distributors are responsible for receiving spans and forwarding them to the appropriate ingesters.  The below configuration
exposes the otlp receiver on port 0.0.0.0:5680.  [This configuration](https://github.com/grafana/tempo/blob/master/example/docker-compose/etc/tempo-s3-minio.yaml) shows how to
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
See [here](https://github.com/grafana/tempo/blob/master/modules/ingester/config.go) for all configuration options.

The ingester is responsible for batching up traces and pushing them to [TempoDB](#storage).

```
ingester:
    lifecycler:
        ring:
            replication_factor: 2   # number of replicas of each span to make while pushing to the backend
    trace_idle_period: 20s          # amount of time before considering a trace complete and flushing it to a block
    max_block_bytes: 1_000_000_000  # maximum size of a block before cutting it
    max_block_duration: 1h          # maximum length of time before cutting a block
```

## Query Frontend
See [here](https://github.com/grafana/tempo/blob/master/modules/frontend/config.go) for all configuration options.

The Query Frontend is responsible for sharding incoming requests for faster processing in parallel (by the queriers).

```
query_frontend:
    query_shards: 10    # number of shards to split the query into
```

## Querier
See [here](https://github.com/grafana/tempo/blob/master/modules/querier/config.go) for all configuration options.

The Querier is responsible for querying the backends/cache for the traceID.

```
querier:
    frontend_worker:
        frontend_address: query-frontend-discovery.default.svc.cluster.local:9095   # the address of the query frontend to connect to, and process queries
```

## [Compactor]
See [here](https://github.com/grafana/tempo/blob/master/modules/compactor/config.go) for all configuration options.

Compactors stream blocks from the storage backend, combine them and write them back.  Values shown below are the defaults.

```
compactor:
    compaction:
        block_retention: 336h               # duration to keep blocks
        compacted_block_retention: 1h       # duration to keep blocks that have been compacted elsewhere
        compaction_window: 1h               # blocks in this time window will be compacted together
        chunk_size_bytes: 10485760          # amount of data to buffer from input blocks
        flush_size_bytes: 31457280          # flush data to backend when buffer is this large
        max_compaction_objects: 1000000     # maximum traces in a compacted block
        retention_concurrency: 10           # Optional. Number of tenants to process in parallel during retention. Default 10.
    ring:
        kvstore:
            store: memberlist       # in a high volume environment multiple compactors need to work together to keep up with incoming blocks.
                                    # this tells the compactors to use a ring stored in memberlist to coordinate.
```

## [Storage]
See [here](https://github.com/grafana/tempo/blob/master/tempodb/config.go) for all configuration options.

The storage block is used to configure TempoDB. It supports S3, GCS, Azure, local file system, and optionally can use Memcached or Redis for increased query performance.  

The following example shows common options.  For platform-specific options refer to the following:
* [Azure](azure/)
* [GCS](gcs/)
* [S3](s3/)
* [Redis](redis/)

```
storage:
    trace:
        backend: gcs                             # store traces in gcs
        gcs:
            bucket_name: ops-tools-tracing-ops   # store traces in this bucket

        blocklist_poll: 5m                       # how often to repoll the backend for new blocks
        blocklist_poll_concurrency: 50           # Optional. Number of blocks to process in parallel during polling. Default 50.
        memcached:                               # optional memcached configuration
            consistent_hash: true
            host: memcached
            service: memcached-client
            timeout: 500ms
        redis:                                   # optional redis configuration 
            endpoint: redis
            timeout: 500ms
        pool:                                    # the worker pool is used primarily when finding traces by id, but is also used by other
            max_workers: 50                      # total number of workers pulling jobs from the queue
            queue_depth: 2000                    # length of job queue
        wal:
            path: /var/tempo/wal                 # where to store the head blocks while they are being appended to
```

## Memberlist
[Memberlist](https://github.com/hashicorp/memberlist) is the default mechanism for all of the Tempo pieces to coordinate with each other.

```
memberlist:
    bind_port: 7946
    join_members:
      - gossip-ring.tracing-ops.svc.cluster.local:7946  # A DNS entry that lists all tempo components.  A "Headless" Cluster IP service in Kubernetes
```
