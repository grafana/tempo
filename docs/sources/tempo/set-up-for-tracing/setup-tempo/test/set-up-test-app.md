---
title: Validate Kubernetes deployment using a test application
menuTitle: Validate Kubernetes deployment
description: Validate your Tempo deployment on Kubernetes.
aliases:
  - ../../../setup/set-up-test-app/ # /docs/tempo/next/setup/set-up-test-app/
weight: 600
---

# Validate Kubernetes deployment using a test application

Once you've set up a Grafana Tempo cluster, you need to write some traces to it and then query the traces from within Grafana.
This procedure uses Tempo in microservices mode.
For example, if you [set up Tempo using the Kubernetes with Tanka procedure](../../deploy/kubernetes/tanka/), then you can use this procedure to validate your set up.

## Before you begin

You'll need:

- Grafana 10.0.0 or higher
- Microservice deployments require the Tempo querier URL, for example: `http://tempo-cluster-query-frontend.tempo.svc.cluster.local:3100/`
- [OpenTelemetry telemetrygen](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/cmd/telemetrygen) for generating tracing data

Refer to [Deploy Grafana on Kubernetes](https://grafana.com/docs/grafana/<GRAANA_VERSION>/setup-grafana/installation/kubernetes/) if you are using Kubernetes.
Otherwise, refer to [Install Grafana](/docs/grafana/<GRAFANA_VERSION>/setup-grafana/installation/) for more information.

## Configure Grafana Alloy to remote-write to Tempo

{{< admonition type="note" >}}
You can skip this section if you have already configured Alloy to send traces to Tempo.
{{< /admonition >}}

[//]: # 'Shared content for best practices for traces'
[//]: # 'This content is located in /tempo/docs/sources/shared/alloy-remote-write-tempo.md'

{{< docs/shared source="tempo" lookup="alloy-remote-write-tempo.md" version="next" >}}

## Create a Grafana Tempo data source

To allow Grafana to read traces from Tempo, you must create a Tempo data source.

1. Navigate to **Connections** > **Data Sources**.

1. Click on **Add data source**.

1. Select **Tempo**.

1. Set the URL to `http://<TEMPO-QUERY-FRONTEND-SERVICE>:<HTTP-LISTEN-PORT>/`, filling in the path to the Tempo query frontend service and the configured HTTP API prefix.
   If you followed [Deploy Tempo with Helm installation example](https://grafana.com/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/deploy/deploy-kubernetes/helm-chart/), the query frontend service's URL looks something like this: `http://tempo-cluster-query-frontend.<NAMESPACE>.svc.cluster.local:3100`

1. Click **Save & Test**.

You should see a message that says `Data source is working`.

## Visualize your data

After you have created a data source, you can visualize your traces in the **Grafana Explore** page.
For more information, refer to [Tempo in Grafana](https://grafana.com/docs/tempo/<TEMPO_VERSION>/introduction/tempo-in-grafana/).

### Use OpenTelemetry `telemetrygen` to generate tracing data

You can use [OpenTelemetry `telemetrygen`](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/cmd/telemetrygen) to generate tracing data to test your Tempo installation.

These instructions use the endpoints for both Grafana Alloy and the Tempo distributor used previously, for example:

- `grafana-alloy.grafana-alloy.svc.cluster.local` for Grafana Alloy
- `tempo-cluster-distributor.tempo.svc.cluster.local` for the Tempo distributor

Update the endpoints if you have altered the endpoint targets.

1. Install `telemetrygen` using the [installation procedure](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/cmd/telemetrygen).
   **NOTE**: You don't need to configure an OpenTelemetry Collector because we're using Grafana Alloy.

2. Generate traces using `telemetrygen`:
   ```bash
   telemetrygen traces --otlp-insecure --rate 20 --duration 5s --otlp-endpoint grafana-alloy.grafana-alloy.svc.cluster.local:4317
   ```
   This configuration sends traces to Alloy for 5 seconds, at a rate of 20 traces per second.

Optionally, you can also send the trace directly to the Tempo database without using Alloy as a collector by using the following:

```bash
telemetrygen traces --otlp-insecure --rate 20 --duration 5s --otlp-endpoint tempo-cluster-distributor.tempo.svc.cluster.local:4317
```

If you're running `telemetrygen` on your local machine, ensure that you first port-forward to the relevant Alloy or Tempo distributor service, for example:

```bash
kubectl port-forward services/grafana-alloy 4317:4317 --namespace grafana-alloy
```

3. Alternatively, you can create a cronjob to send traces periodically based on this template:

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
1. In the `mythical-beasts-deployment.yaml` manifest, alter each `TRACING_COLLECTOR_HOST` environment variable instance value to point to the Grafana Alloy location. For example, based on Alloy installed in the default namespace and with a Helm installation called `test`:
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
1. Once the application is deployed, go to Grafana and select the **Explore** menu item.
1. Select the **Tempo data source** from the list of data sources.
1. Select the `Search` Query type for the data source.
1. Select **Run query**.
1. Traces from the application are displayed in the traces **Explore** panel.
