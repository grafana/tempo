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

* Grafana 10.0.0 or higher
* Microservice deployments require the Tempo querier URL, for example: `http://tempo-cluster-query-frontend.tempo.svc.cluster.local:3100/`
* [OpenTelemetry telemetrygen](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/cmd/telemetrygen) for generating tracing data

Refer to [Deploy Grafana on Kubernetes](/docs/grafana/latest/setup-grafana/installation/kubernetes/#deploy-grafana-on-kubernetes) if you are using Kubernetes.
Otherwise, refer to [Install Grafana](/docs/grafana/latest/installation/) for more information.

## Configure Grafana Alloy to remote-write to Tempo

This section uses a [Grafana Alloy Helm chart](/docs/alloy/<ALLOY_VERSION>/set-up/install/kubernetes/) deployment to send traces to Tempo.

To do this, you need to create a configuration that can be used by Grafana Alloy to receive and export traces in OTLP `protobuf` format.

1. Create a new `values.yaml` file which we'll use as part of the Alloy install.

1. Edit the `values.yaml` file and add the following configuration to it:
   ```yaml
   alloy:
     extraPorts:
       - name: otlp-grpc
         port: 4317
         targetPort: 4317
         protocol: TCP
     configMap:
       create: true
       content: |-
         // Creates a receiver for OTLP gRPC.
         // You can easily add receivers for other protocols by using the correct component
         // from the reference list at: https://grafana.com/docs/alloy/latest/reference/components/
         otelcol.receiver.otlp "otlp_receiver" {
           // Listen on all available bindable addresses on port 4317 (which is the
           // default OTLP gRPC port) for the OTLP protocol.
           grpc {
             endpoint = "0.0.0.0:4317"
           }

           // Output straight to the OTLP gRPC exporter. We would usually do some processing
           // first, most likely batch processing, but for this example we pass it straight
           // through.
           output {
             traces = [
               otelcol.exporter.otlp.tempo.input,
             ]
           }
         }

         // Define an OTLP gRPC exporter to send all received traces to GET.
         // The unique label 'tempo' is added to uniquely identify this exporter.
         otelcol.exporter.otlp "tempo" {
             // Define the client for exporting.
             client {
                 // Send to the locally running Tempo instance, on port 4317 (OTLP gRPC).
                 endpoint = "http://tempo-cluster-distributor.tempo.svc.cluster.local:4317"
                 // Disable TLS for OTLP remote write.
                 tls {
                     // The connection is insecure.
                     insecure = true
                     // Do not verify TLS certificates when connecting.
                     insecure_skip_verify = true
                 }
             }
         }
   ```
   Ensure that you use the specific namespace you've installed Tempo in for the OTLP exporter. In the line:
   ```yaml
   endpoint = "http://tempo-cluster-distributor.tempo.svc.cluster.local:3100"
   ```
   change `tempo` to reference the namespace where Tempo is installed, for example:  `http://tempo-cluster-distributor.my-tempo-namespaces.svc.cluster.local:3100`.

1. Deploy Alloy using Helm:
   ```bash
   helm install -f values.yaml grafana-alloy grafana/alloy
   ```
   If you deploy Alloy into a specific namespace, create the namespace first and specify it to Helm by appending `--namespace=<grafana-alloy-namespace>` to the end of the command.

## Create a Grafana Tempo data source

To allow Grafana to read traces from Tempo, you must create a Tempo data source.

1. Navigate to **Connections** > **Data Sources**.

1. Click on **Add data source**.

1. Select **Tempo**.

1. Set the URL to `http://<TEMPO-QUERY-FRONTEND-SERVICE>:<HTTP-LISTEN-PORT>/`, filling in the path to the Tempo query frontend service, and the configured HTTP API prefix. If you have followed the [Deploy Tempo with Helm installation example](../setup/helm-chart/), the query frontend service's URL looks something like this: `http://tempo-cluster-query-frontend.<namespace>.svc.cluster.local:3100`

1. Click **Save & Test**.

You should see a message that says `Data source is working`.

## Visualize your data

After you have created a data source, you can visualize your traces in the **Grafana Explore** page.
For more information, refer to [Tempo in Grafana]({{< relref "../getting-started/tempo-in-grafana" >}}).

### Use OpenTelemetry `telemetrygen` to generate tracing data

You can use [OpenTelemetry `telemetrygen`](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/cmd/telemetrygen) to generate tracing data to test your Tempo installation.

These instructions use the endpoints for both Grafana Alloy and the Tempo distributor used previously, for example:

* `grafana-alloy.grafana-alloy.svc.cluster.local` for Grafana Alloy
* `tempo-cluster-distributor.tempo.svc.cluster.local` for the Tempo distributor

Update the endpoints if you have altered the endpoint targets.

1. Install `telemetrygen` using the [installation procedure](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/cmd/telemetrygen).
   **NOTE**: You don't need to configure an OpenTelemetry Collector because we're using Grafana Alloy.

2. Generate traces using `telemetrygen`:
   ```bash
   telemetrygen traces --otlp-insecure --rate 20 --duration 5s --otlp-endpoint grafana-alloy.grafana-alloy.svc.cluster.local:4317
   ```
  This configuration sends traces to Grafana Alloy for 5 seconds, at a rate of 20 traces per second.

  Optionally, you can also send the trace directly to the Tempo database without using Grafana Alloy as a collector by using the following:
  ```bash
  telemetrygen traces --otlp-insecure --rate 20 --duration 5s --otlp-endpoint tempo-cluster-distributor.tempo.svc.cluster.local:4317
  ```

  If you're running `telemetrygen` on your local machine, ensure that you first port-forward to the relevant Alloy or Tempo distributor service, for example:
  ```bash
  kubectl port-forward services/grafana-alloy 4317:4317 --namespace grafana-alloy
  ```
3. Alternatively, a cronjob can be created to send traces periodically based on this template:

```
apiVersion: batch/v1
kind: CronJob
metadata:
  name: sample-traces
spec:
  concurrencyPolicy: Forbid
  successfulJobsHistoryLimit: 1
  failedJobsHistoryLimit: 2
  schedule: "0 * * * *"
  jobTemplate:
    spec:
      backoffLimit: 0
      ttlSecondsAfterFinished: 3600
      template:
        spec:
          containers:
          - name: traces
            image: ghcr.io/open-telemetry/opentelemetry-collector-contrib/telemetrygen:v0.96.0
            args:
              - traces
              - --otlp-insecure
              - --rate
              - "20"
              - --duration
              - 5s
              - --otlp-endpoint
              - grafana-alloy.grafana-alloy.svc.cluster.local:4317
          restartPolicy: Never
```

To view the tracing data:

1. Go to Grafana and select **Explore**.

1. Select the **Tempo data source** from the list of data sources.

1. Select the `Search` Query type.

1. Select **Run query**.

1. Confirm that traces are displayed in the traces **Explore** panel. You should see 5 seconds worth of traces, 100 traces in total per run of `telemetrygen`.

### Test your configuration using the Intro to MLTP application

The Intro to MLTP application provides an example five-service application generates data for Tempo, Mimir, Loki, and Pyroscope.
This procedure installs the application on your cluster so you can generate meaningful test data.

1. Navigate to https://github.com/grafana/intro-to-mltp to get the Kubernetes manifests for the Intro to MLTP application.
1. Clone the repository using commands similar to the ones below:
    ```bash
      git clone git+ssh://github.com/grafana/intro-to-mltp
      cp intro-to-mltp/k8s/mythical/* ~/tmp/intro-to-mltp-k8s
    ```
1. Change to the cloned repository: `cd intro-to-mltp/k8s/mythical`
1. In the `mythical-beasts-deployment.yaml` manifest, alter each `TRACING_COLLECTOR_HOST` environment variable instance value to point to the Grafana Alloy location. For example, based on Grafana Alloy installed in the default namespace and with a Helm installation called `test`:
   ```yaml
    	- env:
        ...
        - name: TRACING_COLLECTOR_HOST
          value: grafana-alloy.grafana-alloy.svc.cluster.local
   ```
1. Deploy the Intro to MLTP application. It deploys into the default namespace.
   ```bash
	   kubectl apply -f mythical-beasts-service.yaml,mythical-beasts-persistentvolumeclaim.yaml,mythical-beasts-deployment.yaml
   ```
1. Once the application is deployed, go to Grafana Enterprise and select the **Explore** menu item.
1. Select the **Tempo data source** from the list of data sources.
1. Select the `Search` Query type for the data source.
1. Select **Run query**.
1. Traces from the application will be displayed in the traces **Explore** panel.
