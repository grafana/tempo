---
title: Memcached
---

# Memcached Configuration

Memcached caching is configured in the storage block.

```
storage:
    trace:
        cache: memcached
        memcached:
            host: memcached                             # hostname for memcached service to use.
            service: memcached-client                   # optional. SRV service used to discover memcache servers. (default: memcached)
            addresses: ""                               # (experimental) optional. comma separated addresses list in DNS Service Discovery format. (default: "")
            timeout: 500ms                              # optional. maximum time to wait before giving up on memcached requests. (default: 100ms)
            max_idle_conns: 16                          # optional. maximum number of idle connections in pool. (default: 16)
            update_interval: 1m                         # optional. period with which to poll DNS for memcache servers. (default: 1m)
            consistent_hash: true                       # optional. use consistent hashing to distribute to memcache servers. (default: true)
            circuit_breaker_consecutive_failures: 10    # optional. trip circuit-breaker after this number of consecutive dial failures. (default: 10)
            circuit_breaker_timeout: 10s                # optional. duration circuit-breaker remains open after tripping. (default: 10s)
            circuit_breaker_interval: 10s               # optional. reset circuit-breaker counts after this long. (default: 10s)
``` 