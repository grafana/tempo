---
title: Set up your collector
description: Configure your collector to send traces to Tempo.
weight: 550
aliases:
- /docs/tempo/grafana-alloy
- ../../../configuration/grafana-alloy/ # /docs/tempo/latest/configuration/grafana-alloy/
---

# Set up your collector

An OpenTelemetry Collector distribution provides a way to receive, process, and export telemetry data independent of the vendor.

For production observability, Grafana Labs recommends Grafana Alloy. Alloy is an OpenTelemetry Collector distribution from Grafana Labs. It enables correlation between infrastructure and application observability.

Telemetry data pipelines with an OpenTelemetry Collector distribution offer these benefits:

1. Cost control: A Collector aggregates and drops telemetry data and sends relevant data to reduce costs
2. Scalability and reliability: A Collector buffers and retries sending data, so you don't lose data during connection issues
3. Flexibility: A Collector simplifies configuration and manages data pipelines through enrichment, transformation, redaction, and routing

This document details the following ways to use an OpenTelemetry Collector with Grafana Labs products:

* Grafana Alloy: Use the Grafana Labs-supported OpenTelemetry Collector distribution, recommended for most production environments.
* Grafana Alloy for Kubernetes: Deploy Grafana Alloy in Kubernetes using the Grafana Kubernetes Monitoring Helm chart.
* OpenTelemetry Collector (Upstream): Use the community-supported upstream OpenTelemetry Collector.
* Kubernetes OpenTelemetry Operator: Manage deployments of the upstream OpenTelemetry Collector on Kubernetes.

## Before you begin

Before you set up an OpenTelemetry Collector distribution, ensure you have the following:

* An application or service generating telemetry data, or a data source you want to collect from
* A destination for your telemetry data, such as a Grafana observability stack (Grafana Tempo, Grafana Mimir, Grafana Loki) or Grafana Cloud.
* Network connectivity for the Collector to receive data from sources and send data to destinations

## Grafana Alloy

Grafana Labs maintains and supports Grafana Alloy, which packages various upstream OpenTelemetry Collector components. Alloy provides stability, support, and integration with Grafana Labs products.

## Grafana Alloy for Kubernetes

If you deploy your application in Kubernetes, use the Grafana Kubernetes Monitoring helm chart. This chart supports [Kubernetes Monitoring](https://github.com/grafana/opentelemetry-docs/blob/main/docs/grafana-cloud/monitor-infrastructure/kubernetes-monitoring), Grafana Cloud, and [Application Observability](https://grafana.com/docs/grafana-cloud/monitor-applications/application-observability/).

Refer to [Kubernetes Monitoring with Grafana Alloy](https://github.com/grafana/opentelemetry-docs/blob/main/docs/sources/collector/grafana-alloy-kubernetes) to get started.

## OpenTelemetry Collector

The OpenTelemetry project maintainers and the Cloud Native Computing Foundation maintain the upstream OpenTelemetry Collector. This is a community-supported project.

Refer to the [OpenTelemetry Collector](https://github.com/grafana/opentelemetry-docs/blob/main/docs/sources/collector/opentelemetry-collector) documentation to use the upstream OpenTelemetry Collector with Grafana Labs products.

## Kubernetes OpenTelemetry Operator

If you use the upstream Collector and deploy your application in Kubernetes, you can use the [OpenTelemetry Operator](https://github.com/grafana/opentelemetry-docs/blob/main/docs/sources/instrument/opentelemetry-operator) to instrument your application and send telemetry data to Grafana Cloud without modifying your services.

## Next steps

* [Send and ingest OTLP data in a backend](https://github.com/grafana/opentelemetry-docs/blob/main/docs/sources/ingest)
* [Instrument your applications to send telemetry data to your Collector](https://github.com/grafana/opentelemetry-docs/blob/main/docs/sources/instrument)
* [Gain insights from your telemetry data using Grafana](https://github.com/grafana/opentelemetry-docs/blob/main/docs/sources/insights)
* [Explore the Grafana Alloy GitHub repository](https://github.com/grafana/alloy)
* [Consult the official OpenTelemetry Collector documentation](https://opentelemetry.io/docs/collector/)
