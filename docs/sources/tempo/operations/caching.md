---
title: Improve performance with caching
menuTitle: Improve performance with caching
description: Learn how to improve query performance by using caching.
weight: 600
---

# Improve performance with caching

Tempo uses an external cache to improve query performance.
Tempo supports [Memcached](https://memcached.org/) and [Redis](https://redis.io/) (experimental).

## Cache roles

Tempo caches different types of data, each assigned a **role**.
You configure one or more cache instances under the top-level `cache:` block and assign roles to each instance.
This lets you size and tune each cache independently based on the workload it handles.

| Role | What it caches | Volume |
|---|---|---|
| `bloom` | Bloom filters used for trace ID lookup. | Moderate |
| `trace-id-index` | Trace ID index used to locate traces within blocks. | Moderate |
| `parquet-footer` | Parquet file footer metadata. Useful for both search and trace-by-ID queries. | Low |
| `parquet-column-idx` | Parquet column index sections. | Low |
| `parquet-offset-idx` | Parquet offset index sections. | Low |
| `parquet-page` | Parquet data pages. Caches most Parquet reads. | **High** |
| `frontend-search` | Query-frontend search job results. | Varies |

You can assign multiple roles to a single cache instance, or split high-volume roles (like `parquet-page`) onto a dedicated instance.
For example, you might use a large Memcached pool for `parquet-page` and a smaller one for `bloom` and `parquet-footer`.

For configuration parameters and an example, refer to the [Cache](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#cache) section of the Tempo configuration reference.

For information about search performance, refer to [Tune search performance](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/backend_search/).

## Memcached

Memcached is used by default in the Tanka and Helm examples.
Refer to [Deploy Tempo](https://grafana.com/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/deploy/).

### Connection limit

As a cluster grows in size, the number of instances of Tempo connecting to the cache servers also increases.
By default, Memcached has a connection limit of 1024.
Memcached refuses connections when this limit is surpassed.
You can resolve this issue by increasing the connection limit of Memcached.

You can use the `tempo_memcache_request_duration_seconds_count` metric to observe these errors.
For example, by using the following query:

```promql
sum by (status_code) (
  rate(tempo_memcache_request_duration_seconds_count{}[$__rate_interval])
)
```

This metric is also shown in [the monitoring dashboards](../monitor/) (the left panel):

![QPS and latency of requests to memcached](/media/docs/tempo/caching_memcached_connection_limit.png)

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

In order to decide the values of these configuration parameters, you can use a cache summary command in the [tempo-cli](../tempo_cli/) that
prints a summary of bloom filter shards per day and per compaction level. The result looks something like this:

![Cache summary output](/media/docs/tempo/cache-summary.png)

This image shows the bloom filter shards over 14 days and 6 compaction levels. This can be used to decide the configuration parameters.
