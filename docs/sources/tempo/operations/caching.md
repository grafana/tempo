---
title: Improve performance with caching
menuTitle: Improve performance with caching
description: Learn how to improve query performance by using caching.
weight: 65
---

# Improve performance with caching

Caching is mainly used to improve query performance by storing bloom filters of all backend blocks which are accessed on every query.

Tempo uses an external cache to improve query performance.
Tempo supports [Memcached](https://memcached.org/) and [Redis](https://redis.io/).

For information about search performance, refer to [Tune search performance](https://grafana.com/docs/tempo/latest/operations/backend_search/).

## Memcached

Memcached is one of the cache implementations supported by Tempo.
It's used by default in the Tanka and Helm examples.
Refer to [Deploying Tempo]({{< relref "../setup/deployment" >}}).

### Connection limit

As a cluster grows in size, the number of instances of Tempo connecting to the cache servers also increases.
By default, Memcached has a connection limit of 1024.
Memcached refuses new connections when this limit is surpassed.
You can resolve this issue by increasing the connection limit of Memcached.

You can use the `tempo_memcache_request_duration_seconds_count` metric to observe these errors.
For example, by using the following query:

```promql
sum by (status_code) (
  rate(tempo_memcache_request_duration_seconds_count{}[$__rate_interval])
)
```

This metric is also shown in [the monitoring dashboards]({{< relref "./monitor" >}}) (the left panel):

<p align="center"><img src="../caching_memcached_connection_limit.png" alt="QPS and latency of requests to memcached"></p>

Note that the already open connections continue to function. New connections are refused.

Additionally, Memcached logs the following errors when it can't accept any new requests:

```
accept4(): No file descriptors available
Too many open connections
accept4(): No file descriptors available
Too many open connections
```

When using the [memcached_exporter](https://github.com/prometheus/memcached_exporter), you can observe the number of open connections at `memcached_current_connections`.

## Cache size control

Tempo querier accesses bloom filters of all blocks while searching for a trace.
This essentially mandates the size of cache to be at-least the total size of the bloom filters (the working set).
However, in larger deployments, the working set might be larger than the desired size of cache.
When that happens, eviction rates on the cache grow high, and hit rate drop.

Tempo provides two configuration parameters to filter down on the items stored in cache.

```
        # Min compaction level of block to qualify for caching bloom filter
        # Example: "cache_min_compaction_level: 2"
        [cache_min_compaction_level: <int>]

        # Max block age to qualify for caching bloom filter
        # Example: "cache_max_block_age: 48h"
        [cache_max_block_age: <duration>]
```

Using a combination of these configuration options, you can narrow down on which bloom filters are cached, thereby reducing the
cache eviction rate, and increasing the cache hit rate.

In order to decide the values of these configuration parameters, you can use a cache summary command in the [tempo-cli]({{< relref "./tempo_cli" >}}) that
prints a summary of bloom filter shards per day and per compaction level. The result looks something like this:

<p align="center"><img src="../cache-summary.png" alt="Cache summary"></p>

This image shows the bloom filter shards over 14 days and 6 compaction levels. This can be used to decide the configuration parameters.
