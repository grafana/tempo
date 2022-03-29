---
title: Backend search - serverless GCP setup
weight: 10
---

# Google Cloud Functions

This document will walk you through setting up a Google Cloud Function for serverless backend search.
For more guidance on configuration options for full backend search [check here](./backend_search.md).

1. Build the code package:

    ```bash
    cd ./cmd/tempo-serverless && make build-gcf-zip
    ```

    This will create a ZIP file containing all the code required for 
    the function. The file name will be of the form: `./cloud-functions/tempo-<branch name>-<commit hash>.zip`.
    Here is an example of that name:

    ```bash
    ls cloud-functions/*.zip
    cloud-functions/tempo-serverless-backend-search-297172a.zip
    ```

2. Provision a GCS bucket.

3. Copy the ZIP file into your bucket.

    ```
    gsutil cp cloud-functions/tempo-serverless-backend-search-297172a.zip gs://<newly provisioned gcs bucket>
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
      source_archive_object = "tempo-serverless-backend-search-297172a.zip"
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
      search:
        external_endpoints:
        - <trigger url from console>
    ```
