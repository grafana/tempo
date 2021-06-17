---
title: Caching
weight: 6
---

# Caching

Tempo can use an external cache to improve query performance.
The supported implementations are [Memcached](https://memcached.org/) and [Redis](https://redis.io/). 
Caching is used by the queriers and compactors.

### Memcached

#### Connection limit

As a cluster grows in size, the amount of instances of Tempo connecting to the cache servers will also increase.
By default Memcached has a connection limit of 1024 and if this limit is exceeded new connections will be refused (resulting in HTTP 500 errors).
This is resolved by increasing the connection limit of Memcached.