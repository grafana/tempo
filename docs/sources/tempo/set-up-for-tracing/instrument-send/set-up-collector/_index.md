---
title: Set up your collector
description: Configure your collector to send traces to Tempo.
weight: 550
aliases:
- ../../grafana-alloy/ # /docs/tempo/next/grafana-alloy/
---

# Set up your collector

## Collect and forward traces with auto-instrumentation using Grafana Alloy or OpenTelemetry collectors

You can send data from your application using Grafana Alloy or OpenTelemetry Collector (OTel) collectors.

[Grafana Alloy](https://grafana.com/docs/alloy/latest/) is a vendor-neutral distribution of the OpenTelemetry (OTel) Collector.
Alloy uniquely combines the very best OSS observability signals in the community.
Grafana Alloy uses configuration file written using River.

Alloy is a component that runs alongside your application and periodically gathers tracing data from it.
This method is suitable when you want to collect tracing from applications without modifying their source code.

Here's how it works:

1. Install and configure the collector on the same machine or container where your application is running.
2. The collector periodically retrieves your application's performance tracing data, regardless of the language or technology stack your application is using.
3. The captured traces are then sent to the Tempo server for storage and analysis.

Using a collector provides a hassle-free option, especially when dealing with multiple applications or microservices, allowing you to centralize the profiling process without changing your application's codebase.

## Use Grafana Alloy

Grafana Labs maintains and supports Grafana Alloy, which packages various upstream OpenTelemetry Collector components. Alloy provides stability, support, and integration with Grafana Labs products.

Refer to [Grafana Alloy](/docs/tempo<TEMPO_VERSION>setup-up-for-tracing/setup-up-collector/grafana-alloy) for information about Alloy and it's tracing capabilities.

Refer to [Collect and forward data with Grafana Alloy](https://grafana.com/docs/alloy/<ALLOY_VERSION>/collect/) for examples of collecting data.

### Grafana Alloy for Kubernetes

If you deploy your application in Kubernetes, use the Grafana Kubernetes Monitoring helm chart. This chart supports [Kubernetes Monitoring](https://github.com/grafana/opentelemetry-docs/blob/main/docs/grafana-cloud/monitor-infrastructure/kubernetes-monitoring), Grafana Cloud, and [Application Observability](https://grafana.com/docs/grafana-cloud/monitor-applications/application-observability/).

Refer to [Kubernetes Monitoring with Grafana Alloy](https://github.com/grafana/opentelemetry-docs/blob/main/docs/sources/collector/grafana-alloy-kubernetes) to get started.

### Use the OpenTelemetry Collector

The OpenTelemetry project maintainers and the Cloud Native Computing Foundation maintain the upstream OpenTelemetry Collector. This is a community-supported project.

Refer to the [OpenTelemetry Collector](https://github.com/grafana/opentelemetry-docs/blob/main/docs/sources/collector/opentelemetry-collector) documentation to use the upstream OpenTelemetry Collector with Grafana Labs products.

### Kubernetes OpenTelemetry Operator

If you use the upstream Collector and deploy your application in Kubernetes, you can use the [OpenTelemetry Operator](https://github.com/grafana/opentelemetry-docs/blob/main/docs/sources/instrument/opentelemetry-operator) to instrument your application and send telemetry data to Grafana Cloud without modifying your services.

