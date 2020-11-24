---
title: Configuration
weight: 200
---

# Configuration

This section explains the configuration options for Tempo as well as the details of what they impact. It includes:

  - [Authentication/Server](#authenticationserver)
  - [Distributor](#distributor)
  - [Ingester](#ingester)
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

## [Distributor](https://github.com/grafana/tempo/blob/master/modules/distributor/config.go)
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

## [Ingester](https://github.com/grafana/tempo/blob/master/modules/ingester/config.go)
The ingester is responsible for batching up traces and pushing them to [TempoDB](#storage).

```
ingester:
    lifecycler:
        ring:
            replication_factor: 2   # number of replicas of each span to make while pushing to the backend
    trace_idle_period: 20s          # amount of time before considering a trace complete and flushing it to a block
    traces_per_block: 100000        # maximum number of traces in a block before cutting it
```

## [Compactor](https://github.com/grafana/tempo/blob/master/modules/compactor/config.go)
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
    ring:
        kvstore:
            store: memberlist       # in a high volume environment multiple compactors need to work together to keep up with incoming blocks.
                                    # this tells the compactors to use a ring stored in memberlist to coordinate.
```

## [Storage](https://github.com/grafana/tempo/blob/master/tempodb/config.go)
The storage block is used to configure TempoDB. It supports S3, GCS, local file system, and optionally can use memcached for increased query performance.  

The following example shows common options.  For platform-specific options refer to the following:
* [S3](s3/)

```
storage:
    trace:
        backend: gcs                             # store traces in gcs
        gcs:
            bucket_name: ops-tools-tracing-ops   # store traces in this bucket

        blocklist_poll: 5m                       # how often to repoll the backend for new blocks
        memcached:                               # optional memcached configuration
            consistent_hash: true
            host: memcached
            service: memcached-client
            timeout: 500ms
        redis:                                   # optional redis configuration 
            endpoint: redis
            timeout: 500ms
            db: 0
            expiration: 0s
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
