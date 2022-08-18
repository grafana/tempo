---
title: Google Cloud Storage permissions
weight: 10
---

# Google Cloud Storage permissions

For configuration options, check the storage section on the [configuration]({{< relref "./#storage" >}}) page.

## Permissions
The following authentication methods are supported:
- GCP environment variable `GOOGLE_APPLICATION_CREDENTIALS`

The `(service-)account` that will communicate towards GCS should be assigned to the bucket which will receive the traces and should have the following IAM polices within the bucket:

- `storage.objects.create`
- `storage.objects.delete`
- `storage.objects.get`
- `storage.buckets.get`
