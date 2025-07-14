---
title: Set up monitoring for Tempo
menuTitle: Set up monitoring
description: Set up monitoring for Tempo
weight: 20
---

# Set up monitoring for Tempo 

You can set up monitoring for Tempo using an existing or new cluster.
If you don't have a cluster available, you can use the linked documentation to set up the Tempo, Mimir, and Grafana using Helm or you can use Grafana Cloud.

You can use this procedure to set up monitoring for Tempo running in monolithic (single binary) or microservices modes.

To set up monitoring, you need to:

* Use Grafana Alloy to remote-write to Tempo and set up Grafana to visualize the tracing data by following [Set up a test app](https://grafana.com/docs/tempo/<TEMPO_VERSION>/setup/set-up-test-app/).
* Update your Alloy configuration to scrape metrics to monitor for your Tempo data.

This procedure assumes that you have set up Tempo [using the Helm chart](https://grafana.com/docs/tempo/<TEMPO_VERSION>/setup/helm-chart/) with [Grafana Alloy](https://grafana.com/docs/alloy/<TEMPO_VERSION>/set-up/install/).

The steps outlined below use the Alloy configurations described in [Set up a test application for a Tempo cluster](https://grafana.com/docs/tempo/<TEMPO_VERSION>/setup/set-up-test-app/).

{{< admonition type="note" >}}
Update any instructions in this document for your own deployment.

If you use the [Kubernetes integration Grafana Alloy Helm chart](https://grafana.com/docs/alloy/<ALLOY_VERSION>/set-up/install/kubernetes/), you can use the Kubernetes scrape annotations to automatically scrape Tempo.
You’ll need to add the labels to all of the deployed components.
{{< /admonition >}}

## Before you begin

To configure monitoring using the examples on this page, you’ll need the following running in your Kubernetes environment:

* Tempo instance - For storing traces and emitting metrics ([install using the `tempo-distributed` Helm chart](https://grafana.com/docs/tempo/<TEMPO_VERSION>/setup/helm-chart/))
* Mimir - For storing metrics emitted from Tempo ([install using the `mimir-distributed` Helm chart](https://grafana.com/docs/helm-charts/mimir-distributed/latest/get-started-helm-charts/))
* Grafana - For visualizing traces and metrics ([install on Kubernetes](https://grafana.com/docs/grafana/<GRAFANA_VERSION>/setup-grafana/installation/kubernetes/#deploy-grafana-oss-on-kubernetes))

You can use Grafana Alloy or the OpenTelemetry Collector. This procedure provides examples only for Grafana Alloy.

The rest of this documentation assumes that the Tempo, Grafana, and Mimir instances use the same Kubernetes cluster.

If you are using Grafana Cloud, you can skip the installation sections and set up the [Mimir (Prometheus)](https://grafana.com/docs/grafana-cloud/connect-externally-hosted/data-sources/prometheus/) and [Tempo data sources](https://grafana.com/docs/grafana-cloud/connect-externally-hosted/data-sources/tempo/) in your Grafana instance.

## Use a test app for Tempo to send data to Grafana

Before you can monitor Tempo data, you need to configure Grafana Alloy to send traces to Tempo.

Use [these instructions to create a test application](https://grafana.com/docs/tempo/latest/setup/set-up-test-app/) in your Tempo cluster.
These steps configure Grafana Alloy to `remote-write` to Tempo.
In addition, the test app instructions explain how to configure a Tempo data source in Grafana and view the tracing data.

{{< admonition type="note" >}}
If you already have a Tempo environment, then there is no need to create a test app.
This guide assumes that the Tempo and Grafana Alloy configurations are the same as or based on [these instructions to create a test application](https://grafana.com/docs/tempo/latest/setup/set-up-test-app/), as you'll augment those configurations to enable Tempo metrics monitoring.
{{< /admonition >}}

In these examples, Tempo is installed in a namespace called `tempo`.
Change this namespace name in the examples as needed to fit your own environment.

## Configure Grafana

In your Grafana instance, you'll need:

* [A Tempo data source](https://grafana.com/docs/grafana/<GRAFANA_VERSION>/datasources/tempo/configure-tempo-data-source/) (created in the previous section)
* A [Mimir (Prometheus) data source](https://grafana.com/docs/grafana/<GRAFANA_VERSION>/datasources/prometheus/)


{{< docs/shared source="tempo" lookup="metamonitoring.md" version="<TEMPO_VERSION>" >}}
