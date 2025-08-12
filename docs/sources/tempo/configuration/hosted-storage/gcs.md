---
title: Google Cloud Storage
description: Learn about Google Cloud Storage permissions for Tempo.
aliases:
  - ../../configuration/gcs/ # /docs/tempo/<TEMPO_VERSION>/configuration/gcs/
---

# Google Cloud Storage

For all configuration keys, see the storage section on the [configuration reference](../../#storage) page.

## Before you begin

Ensure that you have a Google Cloud Storage bucket created. Refer to the [Google Cloud Storage documentation](https://cloud.google.com/storage/docs/creating-buckets) for more details.

## Permissions

Authentication methods supported by Tempo:

- Google Cloud Platform environment variable `GOOGLE_APPLICATION_CREDENTIALS`
- Google Cloud Platform Workload Identity

The `(service-)account` that communicates towards GCS should be assigned to the bucket which receives the traces and have the following IAM policies:

- `storage.objects.create`
- `storage.objects.delete`
- `storage.objects.get`
- `storage.buckets.get`
- `storage.objects.list`

## Configuration examples

This section provides configuration examples for Tempo monolithic and distributed modes.

### Tempo monolithic mode

Example configuration for GCS storage for Tempo monolithic mode.

```yaml
tempo:
  storage:
    trace:
      backend: gcs
      gcs:
        bucket_name: my-tempo-bucket
        # optional
        prefix: tempo
        chunk_buffer_size: 10_000_000
        list_blocks_concurrency: 3
        # performance (optional)
        # hedge_requests_at: 500ms
        # hedge_requests_up_to: 2
        # metadata and cache-control (optional)
        # object_cache_control: "no-cache"
        # object_metadata:
        #   env: "prod"
```

### Tempo distributed mode

Example configuration for GCS storage for Tempo distributed mode.

```yaml
storage:
  trace:
    backend: gcs
    gcs:
      bucket_name: my-tempo-bucket
      prefix: tempo

distributor:
  extraArgs:
    - "-config.expand-env=true"

compactor:
  extraArgs:
    - "-config.expand-env=true"

ingester:
  extraArgs:
    - "-config.expand-env=true"

querier:
  extraArgs:
    - "-config.expand-env=true"

queryFrontend:
  extraArgs:
    - "-config.expand-env=true"
```

### Local testing with a GCS emulator

You can use a local emulator such as `fsouza/fake-gcs-server` to test your Tempo configuration locally.
Point Tempo at the emulator endpoint and disable authentication and certificate checks.

```yaml
tempo:
  storage:
    trace:
      backend: gcs
      gcs:
        bucket_name: test-bucket
        endpoint: https://127.0.0.1:4443/storage/v1/
        insecure: true
```

## Performance tuning

- Hedge requests: Set `hedge_requests_at` (for example, `500ms`) and optionally `hedge_requests_up_to` to reduce long-tail latency on reads (most effective on queriers).
- Listing concurrency: Adjust `list_blocks_concurrency` (default 3) to balance speed and resource usage during blocklist polling.

## Object metadata and cache-control

Control how objects are cached and attach custom metadata to objects written by Tempo.

```yaml
gcs:
  object_cache_control: "no-cache"
  object_metadata:
    env: prod
    owner: observability
```

## Troubleshooting

- 401 or 403 errors: Ensure ADC is configured (`GOOGLE_APPLICATION_CREDENTIALS`) or Workload Identity is set up on the Pod, and verify the bucket IAM permissions listed above.
- Slow queries or listing: Increase `list_blocks_concurrency` and consider enabling `hedge_requests_at` on queriers.
