---
title: Redis
---

# Redis Configuration

> Note: Redis support is experimental

Redis caching is configured in the storage block.

```
storage:
    trace:
        cache: redis
        redis:
            endpoint: redis                     # redis endpoint to use when caching.
            timeout: 500ms                      # optional. maximum time to wait before giving up on redis requests. (default 100ms)
            master-name: redis-master           # optional. redis Sentinel master name. (default "") 
            db: 0                               # optional. database index. (default 0)
            expiration: 0s                      # optional. how long keys stay in the redis. (default 0)
            tls-enabled: false                  # optional. enable connecting to redis with TLS. (default false)
            tls-insecure-skip-verify: false     # optional. skip validating server certificate. (default false)
            pool-size: 0                        # optional. maximum number of connections in the pool. (default 0)
            password: ...                       # optional. password to use when connecting to redis. (default "")
            idle-timeout: 0s                    # optional. close connections after remaining idle for this duration. (default 0s)
            max-connection-age: 0s              # optional. close connections older than this duration. (default 0s)
``` 