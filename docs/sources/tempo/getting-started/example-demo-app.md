---
title: Example setups
description: This page provides setup examples of how Tempo can be configured for a sample environment.
aliases:
- /docs/tempo/latest/getting-started/quickstart-tempo/
- /docs/tempo/latest/guides/loki-derived-fields/
weight: 300
---

# Example setups

The following examples show various deployment and configuration options using trace generators so you can get started experimenting with Tempo without an existing application.

For more information about Tempo setup and configuration, see:

* [Set up Tempo]({{< relref "../setup" >}})
* [Tempo configuration]({{< relref "../configuration" >}})

If you are interested in instrumentation, see [Tempo instrumentation]({{< relref "./instrumentation" >}}).

## Docker Compose

The [docker-compose examples](https://github.com/grafana/tempo/tree/main/example/docker-compose) are simpler and designed to show minimal configuration.

Some of the examples include:

- Trace discovery with Loki
- Basic Grafana Alloy/OpenTelemetry Setup
- Various Backends (S3/GCS/Azure)
- [K6 with Traces]({{< relref "./docker-example" >}})
This is a great place to get started with Tempo and learn about various trace discovery flows.

## Helm

The Helm [example](https://github.com/grafana/tempo/tree/main/example/helm) shows a complete microservice based deployment.
There are monolithic mode and microservices examples.

To install Tempo on Kubernetes, use the [Deploy on Kubernetes using Helm](/docs/helm-charts/tempo-distributed/next/) procedure.

## Tanka

To view an example of a complete microservice-based deployment, this [Jsonnet based example](https://github.com/grafana/tempo/tree/main/example/tk) shows a complete microservice based deployment.
There are monolithic mode and microservices examples.

To learn how to set up a Tempo cluster, see [Deploy on Kubernetes with Tanka]({{< relref "../setup/tanka" >}}).

## Introduction to Metrics, Logs and Traces example

The [Introduction to Metrics, Logs and Traces in Grafana](https://github.com/grafana/intro-to-mlt) provides a self-contained environment for learning about Mimir, Loki, Tempo, and Grafana. It includes detailed explanations of each compononent, annotated configurations for each component.

The README.md file has full details on how to quickly download and [start the environment](https://github.com/grafana/intro-to-mlt#running-the-demonstration-environment), including instructions for using Grafana Cloud and the OpenTelemetry Agent.
