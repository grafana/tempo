---
title: Set up a test application for a Tempo cluster
menuTitle: Set up a test application for a Tempo cluster
description: Learn how to set up a test app for your Tempo cluster and visualize data.
weight: 600
---

# Set up a test application for a Tempo cluster

Once you've set up a Grafana Tempo cluster, you need to write some traces to it and then query the traces from within Grafana.
This procedure uses Tempo in microservices mode.
For example, if you [set up Tempo using the Kubernetes with Tanka procedure]({{< relref "./tanka" >}}), then you can use this procedure to test your set up.

## Before you begin

You'll need:

* Grafana 9.0.0 or higher
* Microservice deployments require the Tempo querier URL, for example: `http://query-frontend.tempo.svc.cluster.local:3200`
* [OpenTelemetry telemetrygen](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/cmd/telemetrygen) for generating tracing data

Refer to [Deploy Grafana on Kubernetes](/docs/grafana/latest/setup-grafana/installation/kubernetes/#deploy-grafana-on-kubernetes) if you are using Kubernetes.
Otherwise, refer to [Install Grafana](/docs/grafana/latest/installation/) for more information.

## Set up `remote_write` to your Tempo cluster

To enable writes to your cluster:

1. Add a `remote_write` configuration snippet to the configuration file of an existing Grafana Agent.

   If you do not have an existing traces collector, refer to [Set up with Grafana Agent](/docs/agent/latest/set-up/).
   For Kubernetes, refer to the [Grafana Agent Traces Kubernetes quick start guide](/docs/grafana-cloud/kubernetes-monitoring/agent-k8s/k8s_agent_traces/).

   The example agent Kubernetes ConfigMap configuration below opens many trace receivers (note that the remote write is onto the Tempo cluster using OTLP gRPC):

    ```yaml
    kind: ConfigMap
    metadata:
      name: grafana-agent-traces
    apiVersion: v1
    data:
      agent.yaml: |
        traces:
            configs:
              - batch:
                    send_batch_size: 1000
                    timeout: 5s
                name: default
                receivers:
                    jaeger:
                        protocols:
                            grpc: null
                            thrift_binary: null
                            thrift_compact: null
                            thrift_http: null
                    opencensus: null
                    otlp:
                        protocols:
                            grpc: null
                            http: null
                    zipkin: null
                remote_write:
                  - endpoint: <tempoDistributorServiceEndpoint>
                    insecure: true  # only add this if TLS is not used
                scrape_configs:
                  - bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
                    job_name: kubernetes-pods
                    kubernetes_sd_configs:
                      - role: pod
                    relabel_configs:
                      - action: replace
                        source_labels:
                          - __meta_kubernetes_namespace
                        target_label: namespace
                      - action: replace
                        source_labels:
                          - __meta_kubernetes_pod_name
                        target_label: pod
                      - action: replace
                        source_labels:
                          - __meta_kubernetes_pod_container_name
                        target_label: container
                    tls_config:
                        ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
                        insecure_skip_verify: false
    ```

    If you have followed the [Tanka Tempo installation example]({{< relref "../setup/tanka" >}}), then the `endpoint` value would be:

    ```bash
    distributor.tempo.svc.cluster.local:4317
    ```

1. Apply the ConfigMap with:

    ```bash
    kubectl apply --namespace default -f agent.yaml
    ```

1. Deploy Grafana Agent using the procedures from the relevant instructions above.

## Create a Grafana Tempo data source

To allow Grafana to read traces from Tempo, you must create a Tempo data source.

1. Navigate to **Configuration ≫ Data Sources**.

1. Click on **Add data source**.

1. Select **Tempo**.

1. Set the URL to `http://<TEMPO-HOST>:<HTTP-LISTEN-PORT>/`, filling in the path to your gateway and the configured HTTP API prefix. If you have followed the [Tanka Tempo installation example]({{< relref "../setup/tanka.md" >}}), this will be: `http://query-frontend.tempo.svc.cluster.local:3200/`

1. Click **Save & Test**.

You should see a message that says `Data source is working`.

If you see an error that says `Data source is not working: failed to get trace with id: 0`, check your Grafana version.

To fix the error, [upgrade your Grafana to 9.0 or later](/docs/grafana/latest/setup-grafana/upgrade-grafana/).

## Visualize your data

Once you have created a data source, you can visualize your traces in the **Grafana Explore** page.
For more information, refer to [Tempo in Grafana]({{< relref "../getting-started/tempo-in-grafana" >}}).

### Use OpenTelemetry `telemetrygen` to generate tracing data

Next, you can use [OpenTelemetry `telemetrygen`](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/cmd/telemetrygen) to generate tracing data to test your Tempo installation.

In the following instructions we assume the endpoints for both the Grafana Agent and the Tempo distributor are those described above, for example:
* `grafana-agent-traces.default.svc.cluster.local` for Grafana Agent
* `distributor.tempo.svc.cluster.local` for the Tempo distributor
Replace these appropriately if you have altered the endpoint targets.

1. Install `telemetrygen` using the [installation procedure](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/cmd/telemetrygen).
   **NOTE**: You do not need to configure an OpenTelemetry Collector as we are using the Grafana Agent.

1. Generate traces using `telemtrygen`:
   ```bash
   telemetrygen traces --otlp-insecure --rate 20 --duration 5s grafana-agent-traces.default.svc.cluster.local:4317
   ```
  This configuration sends traces to Grafana Agent for 5 seconds, at a rate of 20 traces per second.

  Optionally, you can also send the trace directly to the Tempo database without using Grafana Agent as a collector by using the following:
  ```bash
  telemetrygen traces --otlp-insecure --rate 20 --duration 5s distributor.tempo.svc.cluster.local:4317
  ```

### View tracing data in Grafana

To view the tracing data:

1. Go to Grafana and select the **Explore** menu item.

1. Select the **Tempo data source** from the list of data sources.

1. Copy the trace ID into the **Trace ID** edit field.

1. Select **Run query**.

1. Confirm that traces are displayed in the traces **Explore** panel. You should see 5 seconds worth of traces, 100 traces in total per run of `telemetrygen`.

