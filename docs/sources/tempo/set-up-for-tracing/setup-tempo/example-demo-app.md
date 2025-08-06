---
title: Example setups
description: This page provides setup examples of how Tempo can be configured for a sample environment.
aliases:
  - ../../guides/loki-derived-fields/ # /docs/tempo/next/guides/loki-derived-fields/
  - ../../getting-started/example-demo-app/ #/docs/tempo/<TEMPO_VERSION>/getting-started/example-demo-app/
weight: 300
---

# Example setups

The following examples show various deployment and configuration options using trace generators so you can get started experimenting with Tempo without an existing application.

For more information about Tempo setup and configuration, see:

- [Set up Tempo](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/)
- [Tempo configuration](/docs/tempo/<TEMPO_VERSION>/configuration/)

If you are interested in instrumentation, refer to [Tempo instrumentation](../../instrument-send/).

## Docker Compose

The [docker-compose examples](https://github.com/grafana/tempo/tree/main/example/docker-compose) are simpler and designed to show minimal configuration.

Some of the examples include:

- Trace discovery with Loki
- Basic Grafana Alloy/OpenTelemetry Setup
- Various Backends (S3/GCS/Azure)
- [K6 with Traces](/docs/tempo/<TEMPO_VERSION>/docker-example/)

This is a great place to get started with Tempo and learn about various trace discovery flows.

## Helm

The Helm [example](https://github.com/grafana/tempo/tree/main/example/helm) shows a complete microservice based deployment.
There are monolithic mode and microservices examples.

To install Tempo on Kubernetes, use the [Deploy on Kubernetes using Helm](https://grafana.com/docs/helm-charts/tempo-distributed/next/) procedure.

## Tanka

To view an example of a complete microservice-based deployment, this [Jsonnet based example](https://github.com/grafana/tempo/tree/main/example/tk) shows a complete microservice based deployment.
There are monolithic mode and microservices examples.

To learn how to set up a Tempo cluster, refer to [Deploy on Kubernetes with Tanka](https://grafana.com/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/deploy/kubernetes/tanka/).

## Introduction to Metrics, Logs, Traces, and Profiles example

The [Introduction to Metrics, Logs, Traces, and Profiles in Grafana](https://github.com/grafana/intro-to-mltp) provides a self-contained environment for learning about Mimir, Loki, Tempo, Pyroscope, and Grafana.
It includes detailed explanations of each component and annotated configurations for each component.

The README.md file explains how to download and [start the environment](https://github.com/grafana/intro-to-mltp#running-the-demonstration-environment), including instructions for using Grafana Cloud and Grafana Alloy collector.
