---
title: Deploy Tempo using Docker Compose
menuTitle: Deploy using Docker Compose
description: Instructions for deploying Tempo using Docker Compose
weight: 400
---

# Deploy Tempo using Docker Compose

The [`example` directory](https://github.com/grafana/tempo/tree/main/example/docker-compose) in the Tempo repository provides a `docker-compose` file that you can use to run Tempo locally with different configurations.

The easiest example to start with is [Local Storage](https://github.com/grafana/tempo/tree/main/example/docker-compose/local/readme.md).
This example runs Tempo as a single binary together with the synthetic-load-generator, to generate traces, and Grafana, to query Tempo.
Data is stored locally on disk.

The following examples showcase specific features or integrations:

- [Grafana Alloy](https://github.com/grafana/tempo/tree/main/example/docker-compose/alloy/readme.md) provides a simple example using the Grafana Alloy as a tracing pipeline.
- [OpenTelemetry Collector](https://github.com/grafana/tempo/tree/main/example/docker-compose/otel-collector/readme.md) is a basic example using the OpenTelemetry Collector as a tracing pipeline.
- [OpenTelemetry Collector Multitenant](https://github.com/grafana/tempo/tree/main/example/docker-compose/otel-collector-multitenant/readme.md) uses the OpenTelemetry Collector in an advanced multi-tenant configuration.

The [Local storage](https://github.com/grafana/tempo/tree/main/example/docker-compose/local/readme.md) example uses the `local` backend, suitable for local testing and development.
