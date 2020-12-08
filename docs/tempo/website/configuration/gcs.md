---
title: Google Cloud Storage (GCS)
---

# GCS Configuration
GCS backend is configured in the storage block.  Tempo requires a dedicated bucket since it maintains a top-level object structure and does not support a custom prefix to nest within a shared bucket.

```
storage:
    trace:
        backend: gcs                                              # store traces in gcs
        s3:
            bucket_name: tempo                                    # store traces in this bucket
            chunk_buffer_size: 10485760                           # optional. buffer size for reads. default = 10MiB
            endpoint: https://storage.googleapis.com/storage/v1/  # optional. api endpoint override
            insecure: false                                       # optional. Set to true to disable authentication 
                                                                  #   and certificate checks.
```
