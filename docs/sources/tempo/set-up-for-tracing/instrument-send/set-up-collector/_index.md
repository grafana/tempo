---
title: Set up your collector
description: Configure your collector to send traces to Tempo.
weight: 550
---

# Set up your collector

You can send data from your application using Grafana Alloy or OpenTelemetry Collector (OTel) collectors.

[Grafana Alloy](https://grafana.com/docs/alloy/<ALLOY_VERSION>/) is a vendor-neutral distribution of the OpenTelemetry (OTel) Collector.
Alloy uniquely combines the very best OSS observability signals in the community.

The [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/) is a vendor-agnostic way to receive, process, and export telemetry data.

The collector is a component that runs alongside your application and periodically gathers tracing data from it.
This method is suitable when you want to collect tracing from applications without modifying their source code.

Here's how it works:

1. Install and configure the collector on the same machine or container where your application is running.
2. The collector periodically retrieves your application's performance tracing data, regardless of the language or technology stack your application is using.
3. The captured traces are then sent to the Tempo server for storage and analysis.

Using a collector provides a hassle-free option, especially when dealing with multiple applications or microservices, allowing you to centralize the tracing process without changing your application's codebase.

## Use Alloy

Grafana Labs maintains and supports Alloy, which packages various upstream OpenTelemetry Collector components. Alloy provides stability, support, and integration with Grafana Labs products.

Refer to the [Alloy documentation](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/instrument-send/set-up-collector/grafana-alloy/) for information about Alloy and it's tracing capabilities.

Refer to [Collect and forward data with Alloy](https://grafana.com/docs/alloy/<ALLOY_VERSION>/collect/) for examples of collecting data.

## Use the OpenTelemetry Collector

The OpenTelemetry project maintainers and the Cloud Native Computing Foundation maintain the upstream OpenTelemetry Collector. This is a community-supported project.

Refer to the [Install the Collector documentation](https://opentelemetry.io/docs/collector/installation/) for instructions on installation.

Refer to the [OpenTelemetry Collector documentation](https://github.com/grafana/opentelemetry-docs/blob/main/docs/sources/collector/opentelemetry-collector) to use the upstream OpenTelemetry Collector with Grafana Labs products.
