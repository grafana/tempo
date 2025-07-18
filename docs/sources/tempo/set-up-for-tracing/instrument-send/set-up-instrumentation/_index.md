---
title: Set up instrumentation
menuTitle: Set up instrumentation
description: Learn how to set up instrumentation for distributed tracing.
weight: 500
---

# Set up instrumentation

Client instrumentation is the first building block to a functioning distributed tracing visualization pipeline.
Client instrumentation is the process of adding instrumentation points in the application that create and offload spans.

Check out these resources for help instrumenting tracing with your favorite languages.
Most of these guides include complete end-to-end examples with Grafana, Loki, Mimir, and Tempo.

## Goal

Add instrumentation to your application to collect metrics, logs, and traces, and send this telemetry data to Tempo.

## Before you begin

* Access to code: You can modify or deploy your application code
* Supported language: Your application uses Java, .NET, JavaScript, Python, PHP, Go, or another language supported by OpenTelemetry
* Data destination: You have access to Tempo

{{< admonition type="note" >}}
For Kubernetes deployments, you don’t need direct code access if you use the OpenTelemetry Operator.
{{< /admonition >}}

## Instrumentation methods

Choose from the following instrumentation approaches:

Grafana distributions: OpenTelemetry SDKs from Grafana with additional features

* [Grafana OpenTelemetry Java](https://grafana.com/docs/opentelemetry/instrument/grafana-java/) (JVM agent, no code changes required, includes support for Scala and Kotlin)
* [Grafana OpenTelemetry .NET](https://grafana.com/docs/opentelemetry/instrument/grafana-dotnet/)

Upstream distributions: OpenTelemetry SDKs maintained by the community

* [OpenTelemetry JavaScript](https://grafana.com/docs/grafana-cloud/monitor-applications/application-observability/instrument/node/)
* [OpenTelemetry Python](https://grafana.com/docs/grafana-cloud/monitor-applications/application-observability/instrument/python/)
* [OpenTelemetry PHP](https://grafana.com/docs/grafana-cloud/monitor-applications/application-observability/instrument/php/)
* [OpenTelemetry Go](https://grafana.com/docs/grafana-cloud/monitor-applications/application-observability/instrument/go/)

Grafana Beyla: Zero-code instrumentation of applications using eBPF technology

* [Grafana Beyla documentation](https://grafana.com/docs/opentelemetry/instrument/beyla/)
* Works with all languages and frameworks
* Requires no code changes
* Requires Linux with Kernel 5.8 or higher with BPF Type Format (BTF) enabled

OpenTelemetry Operator: For Kubernetes deployments

* [OpenTelemetry Operator documentation](https://grafana.com/docs/grafana-cloud/monitor-applications/application-observability/instrument/opentelemetry-operator/)
* Injects OpenTelemetry instrumentation into Kubernetes workloads
* Requires no application code changes

### Comparison of instrumentation methods

| Method | Code changes | Language support | Use case |
|---|---|---|---|
| Grafana OpenTelemetry Java | Not required (JVM agent) | Java | Offers advanced instrumentation features and Grafana support. |
| Grafana OpenTelemetry .NET | Required | .NET | Offers advanced instrumentation features and Grafana support. |
| Upstream OpenTelemetry SDKs | Required | Multiple languages | Provides standard instrumentation with community support. |
| Grafana Beyla | Not required | Any language | Enables quick setup for any language, supports legacy applications, and requires Linux kernel 5.8+ with BTF enabled. |
| OpenTelemetry Operator | Not required | Multiple languages | Manages and injects instrumentation in Kubernetes deployments. |


## Instrumentation frameworks

Most of the popular client instrumentation frameworks have SDKs in the most commonly used programming languages.
You should pick one according to your application needs.

OpenTelemetry has the most active development in the community and may be a better long-term choice.

## OpenTelemetry

A collection of tools, APIs, and SDKs, OpenTelemetry helps engineers instrument, generate, collect, and export telemetry data such as metrics, logs, and traces, to analyze software performance and behavior.
For more information refer to [OpenTelemetry overview](https://grafana.com/oss/opentelemetry/).

### Auto-instrumentation frameworks

OpenTelemetry provides auto-instrumentation agents and libraries of Java, .NET, Python, Go, and JavaScript applications, among others.
For more information, refer for the [OpenTelemetry Instrumentation documentation](https://opentelemetry.io/docs/instrumentation/).

These libraries capture telemetry
information from a client application with minimal manual instrumentation of the codebase.

* [OpenTelemetry Java auto-instrumentation](https://github.com/open-telemetry/opentelemetry-java-instrumentation)
* [OpenTelemetry .NET auto-instrumentation](https://github.com/open-telemetry/opentelemetry-dotnet-instrumentation)
  * [How to configure OpenTelemetry .NET automatic instrumentation with Grafana Cloud](/blog/2023/10/31/how-to-configure-opentelemetry-.net-automatic-instrumentation-with-grafana-cloud)
* [OpenTelemetry Python auto-instrumentation](https://github.com/open-telemetry/opentelemetry-python-contrib)
* [OpenTelemetry Go auto-instrumentation](https://github.com/open-telemetry/opentelemetry-go-instrumentation) and [documentation](https://opentelemetry.io/docs/instrumentation/go/getting-started/)

{{< admonition type="note" >}}
Jaeger client libraries have been deprecated. For more information, refer to the [Deprecating Jaeger clients article](https://www.jaegertracing.io/docs/1.50/client-libraries/#deprecating-jaeger-clients). Jaeger recommends using OpenTelemetry SDKs.
{{< /admonition >}}

### Additional OTel resources

- [Grafana Application Observability](https://grafana.com/docs/grafana-cloud/monitor-applications/application-observability/)
- [OpenTelemetry Go instrumentation examples](https://github.com/open-telemetry/opentelemetry-go-instrumentation/tree/main/examples)
- [OpenTelemetry Language Specific Instrumentation](https://opentelemetry.io/docs/instrumentation/)

## Other instrumentation resources

### Zipkin

- [Zipkin Language Specific Instrumentation](https://zipkin.io/pages/tracers_instrumentation.html)

### Grafana Blog

The Grafana blog periodically features instrumentation posts.

- [How to configure OpenTelemetry .NET automatic instrumentation with Grafana Cloud](https://grafana.com/blog/2023/10/31/how-to-configure-opentelemetry-.net-automatic-instrumentation-with-grafana-cloud)
- [Java Spring Boot Auto-Instrumentation](https://grafana.com/blog/2021/02/03/auto-instrumenting-a-java-spring-boot-application-for-traces-and-logs-using-opentelemetry-and-grafana-tempo/)
- [Go + OpenMetrics Exemplars](https://grafana.com/blog/2020/11/09/trace-discovery-in-grafana-tempo-using-prometheus-exemplars-loki-2.0-queries-and-more/)
- [.NET](https://grafana.com/blog/2021/02/11/instrumenting-a-.net-web-api-using-opentelemetry-tempo-and-grafana-cloud/)
- [Python](https:/grafana.com/blog/2021/05/04/get-started-with-distributed-tracing-and-grafana-tempo-using-foobar-a-demo-written-in-python/)

### Community resources

- [NodeJS](https://github.com/mnadeem/nodejs-opentelemetry-tempo)
- [Java Spring Boot](https://github.com/mnadeem/boot-opentelemetry-tempo)
- [Python](https://github.com/dgzlopes/foobar-demo)
