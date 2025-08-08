---
title: Set up for tracing
menuTitle: Set up for tracing
description: Instructions for setting up Tempo for traces
weight: 300
aliases:
  - ./getting-started/ # /docs/tempo/next/getting-started/
refs:
  examples:
    - pattern: /docs/tempo/
      destination: https://grafana.com/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/example-demo-app/
    - pattern: /docs/enterprise-traces/
      destination: https://grafana.com/docs/enterprise-traces/<ENTERPRISE_TRACES_VERSION>/setup/
  setup:
    - pattern: /docs/tempo/
      destination: https://grafana.com/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/
    - pattern: /docs/enterprise-traces/
      destination: https://grafana.com/docs/enterprise-traces/<ENTERPRISE_TRACES_VERSION>/setup/
  deploy:
    - pattern: /docs/tempo/
      destination: https://grafana.com/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/deploy/
    - pattern: /docs/enterprise-traces/
      destination: https://grafana.com/docs/enterprise-traces/<ENTERPRISE_TRACES_VERSION>/setup/hardware-requirements/
  configure-alloy:
    - pattern: /docs/tempo/
      destination: https://grafana.com/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/instrument-send/set-up-collector/grafana-alloy/
    - pattern: /docs/enterprise-traces/
      destination: https://grafana.com/docs/enterprise-traces/<ENTERPRISE_TRACES_VERSION>/setup/set-up-get-tenants/
---

# Set up for tracing

Grafana Tempo is an open source and high-scale distributed tracing backend.
Tempo lets you search for traces, generate metrics from spans, and link your tracing data with logs and metrics.
Tempo also powers Grafana Cloud Traces and Grafana Enterprise Traces.

Distributed tracing visualizes the lifecycle of a request as it passes through a set of applications.
For more information about traces, refer to [Introduction to traces](https://grafana.com/docs/tempo/<TEMPO_VERSION>/introduction/).

To set up for tracing, you need to:

1. Plan your Tempo deployment to meet your needs. Refer to the [Plan your deployment](./setup-tempo/plan/) documentation.
   Check out the [examples](ref:examples) for ideas on how to get started.
1. Deploy Tempo using the [Set up Tempo documentation](ref:setup).
   Tempo offers different deployment modes. Refer to the [Deployment documentation](ref:deploy) section for more information.
1. Instrument your application or service to emit traces. Use the [Set up instrumentation](http://localhost:3002/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/instrument-send/set-up-instrumentation/) documentation.
1. Set up a collector, like Grafana Alloy or the OpenTelemetry Collector, to offload traces from your application and forward them to Tempo. Refer to the [Set up a collector](./instrument-send/set-up-collector/) documentation.

## Requirements for tracing

Tracing requires client instrumentation, a pipeline for gathering and sending traces, and a backend.

- Client instrumentation enables your application or service to emit traces.
- The pipeline component offloads traces from your application and forwards them to a backend. You can use Grafana Alloy or the OpenTelemetry Collector.
- The backend is the storage and retrieval system for traces, which in this case is Tempo.

Refer to the [Tracing pipeline components](#tracing-pipeline-components) section for more information about these components.

<!-- how to get started with distributed tracing -->

{{< youtube id="zDrA7Ly3ovU" >}}

### Tracing pipeline components

To build a tracing pipeline, you need four major components:
client instrumentation, pipeline, backend, and visualization.

This diagram illustrates a tracing system configuration:

<p align="center"><img src="/media/docs/tempo/intro/tempo-get-started-overview.svg" alt="Tracing pipeline overview"></p>

### Client instrumentation

Client instrumentation (1 in the diagram) is the first building block to a functioning distributed tracing visualization pipeline.
Client instrumentation is the process of adding instrumentation points in the application that
create and offload spans.

{{< admonition type="note" >}}
To learn more about instrumentation, refer to the [Set up instrumentation](http://localhost:3002/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/instrument-send/set-up-instrumentation/) documentation to learn how to instrument your favorite language for distributed tracing.
{{< /admonition >}}

### Pipeline (Grafana Alloy)

After you instrument your application for tracing, the traces are sent
to a backend for storage and visualization. You can build a tracing pipeline that
offloads spans from your application, buffers them, and forwards them to a backend.
Tracing pipelines are optional since most clients can send directly to Tempo.
The pipelines become more critical the larger and more robust your tracing system is.

Grafana Alloy is a service that's deployed close to the application, either on the same node or
within the same cluster (in Kubernetes) to quickly offload traces from the application and forward them to
a storage backend.
Alloy also abstracts features like trace batching to a remote trace backend store, including retries on write failures.

To learn more about Grafana Alloy and how to set it up for tracing with Tempo,
refer to [Grafana Alloy configuration for tracing](ref:configure-alloy).

{{< admonition type="note" >}}
The [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector) / [Jaeger Agent](https://www.jaegertracing.io/docs/latest/deployment/) can also be used at the agent layer.
Refer to [this blog post](/blog/2021/04/13/how-to-send-traces-to-grafana-clouds-tempo-service-with-opentelemetry-collector/)
to see how to use the OpenTelemetry Collector with Tempo.
{{< /admonition >}}

### Backend (Tempo)

The tracing backend stores and retrieves traces on demand.
Grafana Tempo is the distributed tracing backend used to store and query traces.

## Visualize tracing data with Grafana

Grafana and Grafana Cloud have a built-in Tempo data source that you can use to query Tempo and visualize traces.
For more information, refer to the [Tempo data source](https://grafana.com/docs/grafana/<GRAFANA_VERSION>/datasources/tempo/) and the [Tempo in Grafana](https://grafana.com/docs/tempo/<TEMPO_VERSION>/introduction/tempo-in-grafana/) topics.
