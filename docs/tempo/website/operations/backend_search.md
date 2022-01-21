---
title: Backend Search
weight: 9
---

<span style="background-color:#f3f973;">This is an experimental feature.</span>

Backend search can be operationally more complex than traditionally operating Tempo. This is largely due
to how new it is and how our defaults don't support it well. Search is very much a work in progress and 
we expect this to improve as we continue to refine this feature. 

If you are interested in running Tempo with full backed search it is likely your performance will be very poor
unless you make some the of the changes detailed here.

## Configuration Options

Unsurprisingly, the queriers and query-frontends have additional configuration related to backend search. 
Some of the defaults, which are currently tuned for trace by id search, will also need to be tweaked.

**All components**
```
# Required for enabling the experimental functionality
search_enabled: true
```

**Querier**

Without serverless technologies:
```
querier:
  # We need to massively increase the amount of work each querier will attempt
  max_concurrent_queries: 20
```

With serverless technologies (see below):
```
search_enabled: true
querier:
  # At this point the querier is just a proxy to the serverless endpoint. We need
  # to increase this even more to allow massive throughput.
  max_concurrent_queries: 100
  # A list of endpoints to query. This allows for provisioning multiple serverless functions
  # that the load will be evenly spread across.
  search_external_endpoints:
  - https://<serverless endpoint>
```

**Query Frontend**

Check the [documentation]({{< relref "../configuration" >}}) for all configuration options. In particular
check the following config block:

```
query_frontend:
  search:
```

Here we will highlight some tunables to deal with scale.

```
server:
  # At larger scales searching starts to feel more like a batch job. We are committing to improving
  # this but for now you will probably need to up your server timeouts.
  http_server_read_timeout: 2m
  http_server_write_timeout: 2m
query_frontend:
  # If you increase concurrent_jobs we need to increase the queue size per tenant as well or search
  # requests will be 429'ed
  max_outstanding_per_tenant: 2000
  search:
    # At larger scales we need to increase the number of jobs we will attempt simultaneously per search query.
    concurrent_jobs: 2000
```

## Leveraging Serverless

Serverless is not required but with larger loads it's recommended to reduce costs and improve performance. If you find
that you are scaling up your queriers massively and not getting the latencies you would like its time to switch to
serverless. Currently we only have support for Google Cloud Functions, but we will be adding AWS and Azure support in 
coming releases.

### Google Cloud Functions

1. Build the code package:

```
$ cd ./cmd/tempo-serverless && make build-zip
```

This will create a zip file with all code required for the function. The zip will be named `tempo-<commit hash>.zip`.

```
$ ls *.zip
tempo-serverless-2674b233d.zip
```

2. Provision a GCS bucket. This can be created with whatever way is convenient for you: console, cli, terraform, etc.

3. Copy the function zip into your bucket.
```
gsutil cp tempo-serverless-2674b233d.zip gs://<newly provisioned gcs bucket>
```

4. Provision the Google Cloud Function. Again this can be created anyway that works for you. Here is some sample terraform
to help get your started:

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

5. Add the newly created functions as external endpoints to your querier config. The endpoint can be 
retrieved from the trigger tab in Google Cloud Functions:

<p align="center"><img src="../backend_search_cloud_function_trigger.png" alt="Google Cloud Functions trigger tab"></p>

```
querier:
  search_external_endpoints:
  - <trigger url from console>
```