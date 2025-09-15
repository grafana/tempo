---
headless: true
description: Set up Alloy to remote-write traces to Tempo.
labels:
  products:
    - enterprise
    - oss
    - alloy
---

[//]: # 'This file describes how to configure Alloy to remote-write to Tempo.'
[//]: # 'This shared file is included in these locations:'
[//]: # '/tempo/docs/sources/tempo/set-up-for-tracing/validate/set-up-test-app.md'
[//]: # '/tempo/docs/sources/tempo/set-up-for-tracing/instrument-send/set-up-collector/grafana-alloy.md'
[//]: # 'This file is used in the following versions:'
[//]: # 'next'
[//]: # 'latest'
[//]: # 'If you make changes to this file, verify that the meaning and content are not changed in any place where the file is included.'
[//]: # 'Any links should be fully qualified and not relative.'

<!-- Use Alloy to remote-write traces to Tempo shared file. -->

This section uses a [Grafana Alloy Helm chart](/docs/alloy/<ALLOY_VERSION>/set-up/install/kubernetes/) deployment to send traces to Tempo.

To do this, you need to create a configuration that can be used by Alloy to receive and export traces in OTLP `protobuf` format.

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

   change `tempo` to reference the namespace where Tempo is installed, for example: `http://tempo-cluster-distributor.my-tempo-namespaces.svc.cluster.local:3100`.

1. Deploy Alloy using Helm:
   ```bash
   helm install -f values.yaml grafana-alloy grafana/alloy
   ```
   If you deploy Alloy into a specific namespace, create the namespace first and specify it to Helm by appending `--namespace=<grafana-alloy-namespace>` to the end of the command.
