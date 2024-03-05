---
title: Improve performance with caching
menuTitle: Improve performance with caching
weight: 65
---

# Improve performance with caching

Caching is mainly used to improve query performance by storing bloom filters of all backend blocks which are accessed on every query.

Tempo uses an external cache to improve query performance.
The supported implementations are [Memcached](https://memcached.org/) and [Redis](https://redis.io/).

## Memcached

Memcached is one of the cache implementations supported by Tempo.
It is used by default in the Tanka and Helm examples, see [Deploying Tempo]({{< relref "../setup/deployment" >}}).

### Connection limit

As a cluster grows in size, the number of instances of Tempo connecting to the cache servers also increases.
By default, Memcached has a connection limit of 1024. If this limit is surpassed new connections are refused.
This is resolved by increasing the connection limit of Memcached.

These errors can be observed using the `tempo_memcache_request_duration_seconds_count` metric.
For example, by using the following query:

```promql
sum by (status_code) (
  rate(tempo_memcache_request_duration_seconds_count{}[$__rate_interval])
)
```

This metric is also shown in [the monitoring dashboards]({{< relref "monitoring" >}}) (the left panel):

<p align="center"><img src="../caching_memcached_connection_limit.png" alt="QPS and latency of requests to memcached"></p>

Note that the already open connections continue to function, just new connections are refused.

Additionally, Memcached will log the following errors when it can't accept any new requests:

```
accept4(): No file descriptors available
Too many open connections
accept4(): No file descriptors available
Too many open connections
```

When using the [memcached_exporter](https://github.com/prometheus/memcached_exporter), the number of open connections can be observed at `memcached_current_connections`.

## Cache size control

Tempo querier accesses bloom filters of all blocks while searching for a trace. This essentially mandates the size
of cache to be at-least the total size of the bloom filters (the working set) . However, in larger deployments, the
working set might be larger than the desired size of cache. When that happens, eviction rates on the cache grow high,
and hit rate drop. Not nice!

Tempo provides two config parameters in order to filter down on the items stored in cache.

```
        # Min compaction level of block to qualify for caching bloom filter
        # Example: "cache_min_compaction_level: 2"
        [cache_min_compaction_level: <int>]

        # Max block age to qualify for caching bloom filter
        # Example: "cache_max_block_age: 48h"
        [cache_max_block_age: <duration>]
```

Using a combination of these config options, we can narrow down on which bloom filters are cached, thereby reducing our
cache eviction rate, and increasing our cache hit rate. Nice!

In order to decide the values of these config parameters, you can use a cache summary command in the [tempo-cli]({{< relref "tempo_cli" >}}) that
prints a summary of bloom filter shards per day and per compaction level. The result looks something like this:

<p align="center"><img src="../cache-summary.png" alt="Cache summary"></p>

The above image shows the bloom filter shards over 14 days and 6 compaction levels. This can be used to decide the
above configuration parameters.
