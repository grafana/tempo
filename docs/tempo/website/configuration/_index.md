---
title: Configuration
draft: true
weight: 300
---

This document contains most configuration options and details of what they impact. 

### Authentication/Server
Tempo uses the Weaveworks/common server.  See [here](https://github.com/weaveworks/common/blob/master/server/server.go#L45) for all configuration options.

```
auth_enabled: false            # do not require X-Scope-OrgID
server:
  http_listen_port: 3100
```

### [Distributor](https://github.com/grafana/tempo/blob/master/modules/distributor/config.go)
Distributors are responsible for receiving spans and forwarding them to the appropriate ingesters.  The below configuration
exposes the otlp receiver on port 0.0.0.0:5680.  [This configuration](https://github.com/grafana/tempo/blob/master/example/docker-compose/tempo.yaml) shows how to
configure all available receiver options.

```
distributor:
    receivers:
        otlp:
            protocols:
                grpc:
                    endpoint: 0.0.0.0:55680
```

### [Ingester](https://github.com/grafana/tempo/blob/master/modules/ingester/config.go)
The ingester is responsible for batching up traces and pushing them to [TempoDB](#storage).

```
ingester:
    lifecycler:
        ring:
            replication_factor: 2   # number of replicas of each span to make while pushing to the backend
    trace_idle_period: 20s          # amount of time before considering a trace complete and flushing it to a block
    traces_per_block: 100000        # maximum number of traces in a block before cutting it
```

### [Compactor](https://github.com/grafana/tempo/blob/master/modules/compactor/config.go)
Compactors stream blocks from the storage backend, combine them and write them back.

```
compactor:
    compaction:
        block_retention: 336h       # duration to keep blocks
    ring:
        kvstore:
            store: memberlist       # in a high volume environment multiple compactors need to work together to keep up with incoming blocks.
                                    # this tells the compactors to use a ring stored in memberlist to coordinate.
```

### [Storage](https://github.com/grafana/tempo/blob/master/tempodb/config.go)
The storage block is used to configure TempoDB.

```
storage:
    trace:
        backend: gcs                             # store traces in gcs
        gcs:
            bucket_name: ops-tools-tracing-ops   # store traces in this bucket
        maintenance_cycle: 5m                    # how often to repoll the backend for new blocks
        memcached:                               # optional memcached configuration
            consistent_hash: true
            host: memcached
            service: memcached-client
            timeout: 500ms
        pool:                                    # the worker pool is used primarily when finding traces by id, but is also used by other
            max_workers: 50                      # total number of workers pulling jobs from the queue
            queue_depth: 2000                    # length of job queue
        wal:
            path: /var/tempo/wal                 # where to store the head blocks while they are being appended to
```

### Memberlist
[Memberlist](https://github.com/hashicorp/memberlist) is the default mechanism for all of the Tempo pieces to coordinate with each other.

```
memberlist:
    bind_port: 7946
    join_members:
      - gossip-ring.tracing-ops.svc.cluster.local:7946  # A DNS entry that lists all tempo components.  A "Headless" Cluster IP service in Kubernetes
```