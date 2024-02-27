---
title: Google Cloud Storage
description: Learn about Google Cloud Storage permissions for Tempo.
aliases:
  - ../../configuration/gcs/ # /docs/tempo/<TEMPO_VERSION>/configuration/gcs/
---

# Google Cloud Storage

For configuration options, check the storage section on the [configuration]({{< relref "../../configuration#storage" >}}) page.

## Permissions

The following authentication methods are supported:

- Google Cloud Platform environment variable `GOOGLE_APPLICATION_CREDENTIALS`
- Google Cloud Platform Workload Identity

The `(service-)account` that will communicate towards GCS should be assigned to the bucket which will receive the traces and should have the following IAM policies within the bucket:

- `storage.objects.create`
- `storage.objects.delete`
- `storage.objects.get`
- `storage.buckets.get`
- `storage.objects.list`
