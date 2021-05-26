---
title: Getting started
weight: 100
---

# Getting started with Tempo

Distributed tracing visualizes the lifecycle of a request as it passes through
an application. There are a few components that must be configured in order to get a
working distributed tracing visualization.

## 1. Client instrumentation

#### OpenTelemetry Instrumentation SDKs

The first building block to a functioning distributed tracing visualization pipeline
is client instrumentation, which is the process of adding instrumentation points in the application that collects telemetry information. 

SDKs that are available in the most commonly used programming languages are listed below -

* [OpenTemeletry CPP](https://github.com/open-telemetry/opentelemetry-cpp)
* [OpenTemeletry Java](https://github.com/open-telemetry/opentelemetry-java)
* [OpenTemeletry JS](https://github.com/open-telemetry/opentelemetry-js)
* [OpenTemeletry .NET](https://github.com/open-telemetry/opentelemetry-dotnet)
* [OpenTemeletry Lambda](https://github.com/open-telemetry/opentelemetry-lambda)
* [OpenTemeletry Go](https://github.com/open-telemetry/opentelemetry-go)
* [OpenTemeletry Python](https://github.com/open-telemetry/opentelemetry-python)
* [OpenTemeletry Ruby](https://github.com/open-telemetry/opentelemetry-ruby)
* [OpenTemeletry Swift](https://github.com/open-telemetry/opentelemetry-swift)
* [OpenTemeletry Ruby](https://github.com/open-telemetry/opentelemetry-ruby)
* [OpenTemeletry PHP](https://github.com/open-telemetry/opentelemetry-php)
* [OpenTemeletry Rust](https://github.com/open-telemetry/opentelemetry-rust)

#### OpenTelemetry Auto Instrumentation

Some languages have support for auto-instrumentation. These libraries capture telemetry
information from a client application with minimal manual instrumentation of the codebase.

* [OpenTemeletry Java Autoinstrumentation](https://github.com/open-telemetry/opentelemetry-java-instrumentation)
* [OpenTemeletry .NET Autoinstrumentation](https://github.com/open-telemetry/opentelemetry-dotnet-instrumentation)
* [OpenTemeletry Python Autoinstrumentation](https://github.com/open-telemetry/opentelemetry-python-contrib)

> Note: Check out our [instrumentation examples]() to learn how to instrument your
> favourite language for distributed tracing.

## 2. Grafana Agent

Once your application is instrumented for tracing, the next step is to send these traces
to a backend for storage and visualization. The Grafana Agent is a service that is
deployed close to the application, either on the same node or within the same cluster
(in kubernetes) to quickly offload traces from the application and forward them to a storage
backend. It also abstracts features like trace batching and backend routing
away from the client. 

To learn more about the Grafana Agent and how to set it up for tracing with Tempo,
refer to [this blog post](https://grafana.com/blog/2020/11/17/tracing-with-the-grafana-agent-and-grafana-tempo/).

> **Note**: OpenTelemetry Collector / Jaeger Agent can also be used at the agent layer.
> Refer to [this blog post](https://grafana.com/blog/2021/04/13/how-to-send-traces-to-grafana-clouds-tempo-service-with-opentelemetry-collector/)
> to see how the OpenTelemetry Collector can be used with Grafana Cloud Tempo.


## 3. Setting up Tempo Backend

Grafana Tempo is an easy-to-use and high-scale distributed tracing backend used to store and query traces.

Getting started with Tempo is easy.

- If you're looking for a demo application to play around with Tempo, check the [examples with demo app]({{< relref "example-demo-app.md" >}}) topic.
- For an application already instrumented for tracing, [this guide]({{< relref "quickstart-tempo.md" >}}) can help quickly set it up with Tempo.
- For production workloads, refer to the [deployment]({{< relref "../deployment" >}}) section.

> **Note:** The Grafana Agent is already set up to use Tempo. Refer to the [configuration](https://github.com/grafana/agent/blob/main/docs/configuration-reference.md#tempo_config) and [example](https://github.com/grafana/agent/blob/main/example/docker-compose/agent/config/agent.yaml) for details.


## 4. Visualization with Grafana

Grafana has a built in Tempo datasource that can be used to query Tempo and visualize traces.
For more information refer to the [Tempo data source](https://grafana.com/docs/grafana/latest/datasources/tempo/) topic.

For Grafana configuration
