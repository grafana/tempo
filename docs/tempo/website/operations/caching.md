---
title: Caching
weight: 6
---

# Caching

Caching is mainly used to improve query performance by storing bloom filters of all backend blocks which are accessed on every query.

Tempo uses an external cache to improve query performance.
The supported implementations are [Memcached](https://memcached.org/) and [Redis](https://redis.io/). 

### Memcached

#### Connection limit

As a cluster grows in size, the number of instances of Tempo connecting to the cache servers also increases.
By default, Memcached has a connection limit of 1024. If this limit is surpassed new connections are refused.
This is resolved by increasing the connection limit of Memcached.

These errors can be observed using the `cortex_memcache_request_duration_seconds_count` metric.
For example, by using the following query:

```promql
sum by (status_code) (
  rate(cortex_memcache_request_duration_seconds_count{}[$__rate_interval])
)
```

This metric is also shown in [the monitoring dashboards](../monitoring) (the left panel):

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
