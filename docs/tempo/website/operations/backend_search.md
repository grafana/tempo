---
title: Backend search
weight: 9
---

# Backend search

<span style="background-color:#f3f973;">Search is an experimental feature.</span>

Backend search is not yet mature.
It can therefore be operationally more complex.
The defaults do not yet support Tempo well.

Search of the backend datastore will likely exhibit poor performance
unless you make some the of the changes detailed here.

## Configuration

Queriers and query frontends have additional configuration related
to search of the backend datastore. 
Some defaults are currently tuned for a search by trace ID.

### All components

```
# Enable search functionality
search_enabled: true
```

### Querier

Without serverless technologies:

```
querier:
  # Greatly increase the amount of work each querier will attempt
  max_concurrent_queries: 20
```

With serverless technologies:

```
search_enabled: true
querier:
  # The querier is only a proxy to the serverless endpoint.
  # Increase this greatly to permit needed throughput.
  max_concurrent_queries: 100

  # A list of endpoints to query. Load will be spread evenly across
  # these multiple serverless functions.
  search_external_endpoints:
  - https://<serverless endpoint>
```

### Query frontend

[Query frontend](../../configuration#query-frontend) lists all configuration
options. 

These suggestions will help deal with scaling issues.

```
server:
  # At larger scales, searching starts to feel more like a batch job.
  # Increase the server timeout intervals.
  http_server_read_timeout: 2m
  http_server_write_timeout: 2m

query_frontend:
  # When increasing concurrent_jobs, also increase the queue size per tenant,
  # or search requests will be cause 429 errors.
  max_outstanding_per_tenant: 2000

  search:
    # At larger scales, increase the number of jobs attempted simultaneously,
    # per search query.
    concurrent_jobs: 2000
```

## Serverless environment

Serverless is not required,
but with larger loads,
serverless is recommended to reduce costs and improve performance.
If you find that you are scaling up your quantity of queriers,
yet are not acheiving the latencies you would like,
switch to serverless.
Tempo has support for Google Cloud Functions.

### Google Cloud Functions

1. Build the code package:

    ```bash
    cd ./cmd/tempo-serverless && make build-zip
    ```

    This will create a ZIP file containing all the code required for 
    the function. The file name will be of the form: `tempo-<commit hash>.zip`.
    Here is an example of that name:

    ```bash
    ls *.zip
    tempo-serverless-2674b233d.zip
    ```

2. Provision a GCS bucket.

3. Copy the ZIP file into your bucket.

    ```
    gsutil cp tempo-serverless-2674b233d.zip gs://<newly provisioned gcs bucket>
    ```

4. Provision the Google Cloud Function. This example uses Terraform:

    ```
    locals {
      // this can be increased if you would like to use multiple functions
      count = 1
    }
    
    resource "google_cloudfunctions_function" "function" {
      count = local.count
    
      name        = "<function name>-${count.index}"
      description = "Tempo Search Function"
      runtime     = "go116"
    
      available_memory_mb   = 1024
      source_archive_bucket = <GCS bucket created above>
      source_archive_object = "tempo-serverless-2674b233d.zip"
      trigger_http          = true
      entry_point           = "Handler"
      ingress_settings      = "ALLOW_INTERNAL_ONLY"
      min_instances         = 1
    
      // Tempo serverless functions are configured via environment variables
      environment_variables = {
        "TEMPO_GCS_BUCKET_NAME"          = "<GCS bucket name backing your Tempo instance>"
        "TEMPO_BACKEND"                  = "gcs"
        "TEMPO_GCS_HEDGE_REQUESTS_AT"    = "400ms"
        "TEMPO_GCS_HEDGE_REQUESTS_UP_TO" = "2"
      }
    }
    ```

5. Add the newly-created functions as external endpoints in your querier
configuration.
The endpoint can be retrieved from the trigger tab in Google Cloud Functions:

    <p align="center"><img src="../backend_search_cloud_function_trigger.png" alt="Google Cloud Functions trigger tab"></p>

    ```
    querier:
      search_external_endpoints:
      - <trigger url from console>
    ```
