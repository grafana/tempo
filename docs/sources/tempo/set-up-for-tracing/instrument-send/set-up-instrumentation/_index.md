---
title: Set up instrumentation
menuTitle: Set up instrumentation
description: Learn how to set up instrumentation for distributed tracing.
weight: 500
---

# Set up instrumentation

Client instrumentation is the first building block to a functioning distributed tracing visualization pipeline.
Client instrumentation is the process of adding instrumentation points in the application that create and offload spans.

When you set up instrumentation, you:

1. Choose an instrumentation method to use with your application
1. Instrument your application to generate traces

After you set up instrumentation, you can [set up a collector](../set-up-collector/) to receive traces from your application.
Refer to [About instrumentation](../about-instrumentation/) for more information about instrumentation and how it works.

## Choose an instrumentation method

When sending traces to Tempo, you can choose between four methods:

- Auto-instrumentation applies instrumentation automatically using agents or middleware, without code changes.
- Zero-code instrumentation, which uses eBPF technology to instrument applications without code changes. [Grafana Beyla](https://grafana.com/docs/beyla/latest/) is an example of a zero-code instrumentation tool.
- Manual instrumentation involves adding code to create spans and traces, giving full control over collected data.
- Hybrid instrumentation, which combines auto and manual instrumentation, using automatic for most code and manual for custom tracing logic.

Refer to [About instrumentation](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/instrument-send/about-instrumentation/) for more information.

If you are using OTel or Alloy, refer to [Instrument an application with OpenTelemetry](https://grafana.com/docs/opentelemetry/instrument/) for more information.
These instructions are specific to Grafana Cloud, but can be adapted for self-hosted Tempo.

## Instrument your app

Most of the popular client instrumentation frameworks have SDKs in the most commonly used programming languages.
You should pick one according to your application needs.

OpenTelemetry has the most active development in the community and may be a better long-term choice.

Popular instrumentation frameworks include:

- [OpenTelemetry](https://opentelemetry.io/docs/concepts/instrumenting/)
- [Zipkin](https://zipkin.io/pages/tracers_instrumentation)
- [Grafana Beyla](https://grafana.com/docs/beyla/)

### Instrument using OpenTelemetry

A collection of tools, APIs, and SDKs, OpenTelemetry helps engineers instrument, generate, collect, and export telemetry data such as metrics, logs, and traces, to analyze software performance and behavior.
For more information refer to [OpenTelemetry overview](https://grafana.com/oss/opentelemetry/).

If you are using OTel with Grafana Cloud, refer to [Instrument an application with OpenTelemetry](https://grafana.com/docs/opentelemetry/instrument/) for more information.

#### Use OpenTelemetry auto-instrumentation frameworks

OpenTelemetry provides auto-instrumentation agents and libraries of Java, .NET, Python, Go, and JavaScript applications, among others.
For more information, refer for the [OpenTelemetry Instrumentation documentation](https://opentelemetry.io/docs/instrumentation/).

These libraries capture telemetry
information from a client application with minimal manual instrumentation of the codebase.

- [OpenTelemetry Java auto-instrumentation](https://github.com/open-telemetry/opentelemetry-java-instrumentation)
- [OpenTelemetry .NET auto-instrumentation](https://github.com/open-telemetry/opentelemetry-dotnet-instrumentation)
  - [How to configure OpenTelemetry .NET automatic instrumentation with Grafana Cloud](/blog/2023/10/31/how-to-configure-opentelemetry-.net-automatic-instrumentation-with-grafana-cloud)
- [OpenTelemetry Python auto-instrumentation](https://github.com/open-telemetry/opentelemetry-python-contrib)
- [OpenTelemetry Go auto-instrumentation](https://github.com/open-telemetry/opentelemetry-go-instrumentation) and [documentation](https://opentelemetry.io/docs/instrumentation/go/getting-started/)

{{< admonition type="note" >}}
Jaeger client libraries have been deprecated. For more information, refer to the [Deprecating Jaeger clients article](https://www.jaegertracing.io/docs/1.50/client-libraries/#deprecating-jaeger-clients). Jaeger recommends using OpenTelemetry SDKs.
{{< /admonition >}}

#### Additional OTel resources

- [Grafana Application Observability](https://grafana.com/docs/grafana-cloud/monitor-applications/application-observability/)
- [OpenTelemetry Go instrumentation examples](https://github.com/open-telemetry/opentelemetry-go-instrumentation/tree/main/examples)
- [OpenTelemetry Language Specific Instrumentation](https://opentelemetry.io/docs/instrumentation/)

### Instrument with Zipkin auto-instrumentation

Zipkin is a distributed tracing system that helps gather timing data needed to troubleshoot latency problems in microservice architectures.

Refer to the [Zipkin Language Specific Instrumentation](https://zipkin.io/pages/tracers_instrumentation.html) documentation for more information.

If you are using Zipkin with Alloy, refer to the Zipkin receiver, [otelcol.receiver.zipkin documentation](https://grafana.com/docs/alloy/<ALlOY_VERSION>/reference/components/otelcol/otelcol.receiver.zipkin/).

In addition, you can use Zipkin to instrument a library, refer to [Instrumenting a library with Zipkin](https://zipkin.io/pages/instrumenting.html)

Within Grafana, you can also use these Zipkin specific features:

- [Zipkin data source](https://grafana.com/docs/grafana/latest/datasources/zipkin/)
- [Monitor Zipkin with Prometheus and Grafana Cloud](https://grafana.com/docs/grafana-cloud/send-data/metrics/metrics-prometheus/prometheus-config-examples/the-zipkin-community-zipkin/)

### Instrument with Grafana Beyla

Grafana Beyla is an eBPF-based application zero-code instrumentation tool to easily get started with Application Observability. Beyla uses eBPF to automatically inspect application executables and the OS networking layer, and capture trace spans related to web transactions and Rate Errors Duration (RED) metrics for Linux HTTP/S and gRPC services. All data capture occurs without any modifications to application code or configuration.

Refer to [Set up Beyla](https://grafana.com/docs/beyla/<BEYLA_VERSION>/setup/) for information about how to instrument using Beyla.

<!-- update these blog links
## Grafana Blog

The Grafana blog periodically features instrumentation posts.

- [How to configure OpenTelemetry .NET automatic instrumentation with Grafana Cloud](https://grafana.com/blog/2023/10/31/how-to-configure-opentelemetry-.net-automatic-instrumentation-with-grafana-cloud)
- [Java Spring Boot Auto-Instrumentation](https://grafana.com/blog/2021/02/03/auto-instrumenting-a-java-spring-boot-application-for-traces-and-logs-using-opentelemetry-and-grafana-tempo/)
- [Go + OpenMetrics Exemplars](https://grafana.com/blog/2020/11/09/trace-discovery-in-grafana-tempo-using-prometheus-exemplars-loki-2.0-queries-and-more/)
- [.NET](https://grafana.com/blog/2021/02/11/instrumenting-a-.net-web-api-using-opentelemetry-tempo-and-grafana-cloud/)
- [Python](https:/grafana.com/blog/2021/05/04/get-started-with-distributed-tracing-and-grafana-tempo-using-foobar-a-demo-written-in-python/)
-->

### Community resources

- [NodeJS](https://github.com/mnadeem/nodejs-opentelemetry-tempo)
- [Java Spring Boot](https://github.com/mnadeem/boot-opentelemetry-tempo)
- [Python](https://github.com/dgzlopes/foobar-demo)

## Next steps

After you set up instrumentation, you can [set up a collector](../set-up-collector/) to receive traces from your application.
