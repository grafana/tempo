---
title: Caching
weight: 6
---

# Caching

Caching is used by the queriers and compactors.
Tempo can use an external cache to improve query performance.
The supported implementations are [Memcached](https://memcached.org/) and [Redis](https://redis.io/). 

### Memcached

#### Connection limit

As a cluster grows in size, the number of instances of Tempo connecting to the cache servers also increases.
By default, Memcached has a connection limit of 1024. If this limit is surpassed new connections are refused (resulting in HTTP 500 errors).
This is resolved by increasing the connection limit of Memcached.
