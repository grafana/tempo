---
title: Get started
menuTitle: Get started with Grafana Tempo
weight: 150
---

# Get started with Grafana Tempo

Distributed tracing visualizes the lifecycle of a request as it passes through
a set of applications.
 To build  a tracing pipeline, you need four major components:
client instrumentation, pipeline, backend, and visualization.

This diagram illustrates a tracing system configuration:

<p align="center"><img src="getting-started.png" alt="Tracing pipeline overview"></p>

> **Note:** You can use [Grafana Cloud](https://grafana.com/products/cloud/features/#cloud-dashboards-grafana) to avoid installing, maintaining, and scaling your own instance of Grafana Tempo. The free forever plan includes 50GB of free traces. [Create an account to get started](https://grafana.com/auth/sign-up/create-user?pg=docs-tempo&plcmt=in-text).

## Client instrumentation

Client instrumentation (1 in the diagram) is the first building block to a functioning distributed tracing visualization pipeline.
Client instrumentation is the process of adding instrumentation points in the application that
create and offload spans.

Most of the popular client instrumentation frameworks
have SDKs in the most commonly used programming languages.
You should pick one according to your application needs.

* [OpenTracing/Jaeger](https://www.jaegertracing.io/docs/latest/client-libraries/)
* [Zipkin](https://zipkin.io/pages/tracers_instrumentation)
* [OpenTelemetry](https://opentelemetry.io/docs/concepts/instrumenting/)

### OpenTelemetry auto-instrumentation

Some languages have support for auto-instrumentation. These libraries capture telemetry
information from a client application with minimal manual instrumentation of the codebase.

* [OpenTelemetry Java auto-instrumentation](https://github.com/open-telemetry/opentelemetry-java-instrumentation)
* [OpenTelemetry .NET auto-instrumentation](https://github.com/open-telemetry/opentelemetry-dotnet-instrumentation)
* [OpenTelemetry Python auto-instrumentation](https://github.com/open-telemetry/opentelemetry-python-contrib)

> **Note**: Check out the [instrumentation references]({{< relref "./instrumentation" >}}) to learn how to instrument your
> favorite language for distributed tracing.

## Pipeline (Grafana Agent)

Once your application is instrumented for tracing, the traces need to be sent
to a backend for storage and visualization. You can build a tracing pipeline that
offloads spans from your application, buffers them, and eventually forwards them to a backend.
Tracing pipelines are optional (most clients can send directly to Tempo), but the pipelines
become more critical the larger and more robust your tracing system is.

The Grafana Agent is a service that is deployed close to the application, either on the same node or
within the same cluster (in Kubernetes) to quickly offload traces from the application and forward them to
a storage backend.
The Grafana Agent also abstracts features like trace batching and backend routing away from the client.

To learn more about the Grafana Agent and how to set it up for tracing with Tempo,
refer to [this blog post](https://grafana.com/blog/2020/11/17/tracing-with-the-grafana-cloud-agent-and-grafana-tempo/).

> **Note**: The [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector) / [Jaeger Agent](https://www.jaegertracing.io/docs/latest/deployment/) can also be used at the agent layer.
> Refer to [this blog post](https://grafana.com/blog/2021/04/13/how-to-send-traces-to-grafana-clouds-tempo-service-with-opentelemetry-collector/)
> to see how the OpenTelemetry Collector can be used with Grafana Cloud Tempo.

## Backend (Tempo)

Grafana Tempo is an easy-to-use and high-scale distributed tracing backend used to store and query traces.
The tracing backend stores and retrieves traces on demand.

Getting started with Tempo is easy.

First, check out the [examples]({{< relref "example-demo-app.md" >}}) for ideas on how to get started with Tempo.

Next, review the [setup documentation]({{< relref "../setup/" >}}) for step-by-step instructions on setting up a Tempo cluster and creating a test app.

For production workloads, refer to the [deployment]({{< relref "../operations/deployment" >}}) section.

> **Note:** The Grafana Agent is already set up to use Tempo. Refer to the [configuration](https://grafana.com/docs/agent/latest/configuration/traces-config/) and [example](https://github.com/grafana/agent/blob/main/example/docker-compose/agent/config/agent.yaml) for details.


## Visualization (Grafana)

Grafana has a built-in Tempo data source that can be used to query Tempo and visualize traces.
For more information, refer to the [Tempo data source](https://grafana.com/docs/grafana/latest/datasources/tempo/) and the [Tempo in Grafana]({{< relref "./tempo-in-grafana/" >}}) topics.

See [querying configuration documentation]({{< relref "../configuration/querying" >}}) for details about Grafana configuration.
