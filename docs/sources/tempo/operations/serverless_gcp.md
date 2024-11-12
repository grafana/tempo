---
title: Search with Google Cloud Run
description: Learn how to set up Google Cloud Run for serverless backend search.
weight: 93
alias:
- /docs/tempo/latest/operations/backend_search/serverless_gcp/
- /docs/tempo/latest/operations/serverless_gcp/
---

# Search with Google Cloud Run

{{< admonition type="caution" >}}
The Tempo serverless feature is now deprecated and will be removed in an upcoming release.
{{< /admonition >}}

This document walks you through setting up a Google Cloud Run for serverless backend search.
For more guidance on configuration options for full backend search [check here]({{< relref "./backend_search" >}}).

1. Build the docker image:

    ```bash
    cd ./cmd/tempo-serverless && make build-docker-gcr
    ```

    This will create the Docker container image to be deployed to Google Cloud Run.
    The docker image will be named: `tempo-serverless:latest` and `tempo-serverless:<branch>-<commit hash>`.
    Here is an example of that name:

    ```bash
    $ docker images | grep tempo-serverless
    tempo-serverless                                                           cloud-run-3be4efa               146c9d9fa63c   58 seconds ago   47.9MB
    tempo-serverless                                                           latest                          146c9d9fa63c   58 seconds ago   47.9MB
    ```

1. Push the image to a Google Container Registry repo.

1. Provision the Google Cloud Run service. This example uses Terraform. Configuration values
   should be adjusted to meet the needs of your installation.

    ```
    locals {
      // this can be increased if you would like to use multiple functions
      count = 1
    }

    resource "google_cloud_run_service" "run" {
      count = local.count

      name     = "<service name>"
      location = "<appropriate region>"

      metadata {
        annotations = {
            "run.googleapis.com/ingress"      = "internal",     # this annotation can be used to limit connectivity to the service
        }
      }

      template {
        metadata {
          annotations = {
              "autoscaling.knative.dev/minScale"                   = "1",
              "autoscaling.knative.dev/maxScale"                   = "1000",
              "autoscaling.knative.dev/panic-threshold-percentage" = "110.0",  # default 200.0. how aggressively to go into panic mode and start scaling heavily
              "autoscaling.knative.dev/window"                     = "10s",    # default 60s. window over which to average metrics to make scaling decisions
          }
        }
        spec {
          container_concurrency = 4
          containers {
            image = "<container image created above>"
            resources {
              limits = {
                  cpu = "2"
                  memory = "1Gi"
              }
            }
            env {
              name = "TEMPO_GCS_BUCKET_NAME"
              value = "<gcs bucket where tempo data is stored>"
            }
            env {
              name = "TEMPO_BACKEND"
              value = "gcs"
            }
            env {
              name = "TEMPO_GCS_HEDGE_REQUESTS_AT"
              value = "400ms"
            }
            env {
              name = "TEMPO_GCS_HEDGE_REQUESTS_UP_TO"
              value = "2"
            }
            env {
              name = "GOGC"
              value = "400"
            }
          }
        }
      }

      traffic {
        percent         = 100
        latest_revision = true
      }
    }
    ```

1. Add the newly-created cloud run service as external endpoints in your querier
configuration. The endpoint can be retrieved from the **Details** tab in Google Cloud Run:

    ```
    querier:
      search:
        external_endpoints:
        - <trigger url from console>
    ```
