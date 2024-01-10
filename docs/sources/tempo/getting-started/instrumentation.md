---
title: Instrument for distributed tracing
menuTitle: Instrument for tracing
description: Client instrumentation is the first building block to a functioning distributed tracing visualization pipeline.
aliases:
- /docs/tempo/latest/guides/instrumentation/
weight: 200
---

# Instrument for distributed tracing

Client instrumentation is the first building block to a functioning distributed tracing visualization pipeline.
Client instrumentation is the process of adding instrumentation points in the application that create and offload spans.

Check out these resources for help instrumenting tracing with your favorite languages.
Most of these guides include complete end-to-end examples with Grafana, Loki, Mimir, and Tempo.

## Instrumentation frameworks

Most of the popular client instrumentation frameworks have SDKs in the most commonly used programming languages.
You should pick one according to your application needs.

OpenTelemetry has the most active development in the community and may be a better long-term choice.

* [OpenTelemetry](https://opentelemetry.io/docs/concepts/instrumenting/)
* [Zipkin](https://zipkin.io/pages/tracers_instrumentation)
* [OpenTracing/Jaeger](https://www.jaegertracing.io/docs/latest/client-libraries/) (deprecated)

## OpenTelemetry

A collection of tools, APIs, and SDKs, [OpenTelemetry](/docs/opentelemetry) helps engineers instrument, generate, collect, and export telemetry data such as metrics, logs, and traces, to analyze software performance and behavior.

### Auto-instrumentation frameworks

OpenTelemetry provides auto-instrumentation agents and libraries of Java, .Net, Python, Go, and JavaScript applications, among others.
For more information, refer for the [OpenTelemetry Instrumentation documentation](https://opentelemetry.io/docs/instrumentation/).

These libraries capture telemetry
information from a client application with minimal manual instrumentation of the codebase.

* [OpenTelemetry Java auto-instrumentation](https://github.com/open-telemetry/opentelemetry-java-instrumentation) and [documentation](/docs/opentelemetry/instrumentation/java/)
    - [Java auto-instrumentation with Java and OTel Java Agent](/docs/opentelemetry/instrumentation/java/javaagent/)
    - [Automatic instrumentation of Spring Boot 3.x applications with Grafana OpenTelemetry Starter](/docs/opentelemetry/instrumentation/java/spring-starter/)
* [OpenTelemetry .NET auto-instrumentation](https://github.com/open-telemetry/opentelemetry-dotnet-instrumentation)
  * [How to configure OpenTelemetry .NET automatic instrumentation with Grafana Cloud](/blog/2023/10/31/how-to-configure-opentelemetry-.net-automatic-instrumentation-with-grafana-cloud)
* [OpenTelemetry Python auto-instrumentation](https://github.com/open-telemetry/opentelemetry-python-contrib)
* [OpenTelemetry Go auto-instrumentation](https://github.com/open-telemetry/opentelemetry-go-instrumentation) and [documentation](https://opentelemetry.io/docs/instrumentation/go/getting-started/)

### Additional OTel resources

- [Java HTTP Metrics from OpenTelemetry Traces](/docs/opentelemetry/instrumentation/java/metrics-from-traces/)
- [OpenTelemetry documentation at Grafana](/docs/opentelemetry)
- [OpenTelemetry Go instrumentation examples](https://github.com/open-telemetry/opentelemetry-go/tree/main/example)
- [OpenTelemetry Language Specific Instrumentation](https://opentelemetry.io/docs/instrumentation/)

## Other instrumentation resources

### Zipkin

- [Zipkin Language Specific Instrumentation](https://zipkin.io/pages/tracers_instrumentation.html)

### Jaeger

{{% admonition type="note" %}}
Jaegar client libraries have been deprecated. For more information, refer to the [Deprecating Jaeger clients article](https://www.jaegertracing.io/docs/1.50/client-libraries/#deprecating-jaeger-clients). Jaegar now recommends using OpenTelemetry SDKs.
{{% /admonition %}}

- [Jaeger Language Specific Instrumentation](https://www.jaegertracing.io/docs/latest/client-libraries/)

## Grafana Blog

The Grafana blog periodically features instrumentation posts.

- [How to configure OpenTelemetry .NET automatic instrumentation with Grafana Cloud](/blog/2023/10/31/how-to-configure-opentelemetry-.net-automatic-instrumentation-with-grafana-cloud)
- [Java Spring Boot Auto-Instrumentation](/blog/2021/02/03/auto-instrumenting-a-java-spring-boot-application-for-traces-and-logs-using-opentelemetry-and-grafana-tempo/)
- [Go + OpenMetrics Exemplars](/blog/2020/11/09/trace-discovery-in-grafana-tempo-using-prometheus-exemplars-loki-2.0-queries-and-more/)
- [.NET](/blog/2021/02/11/instrumenting-a-.net-web-api-using-opentelemetry-tempo-and-grafana-cloud/)
- [Python](/blog/2021/05/04/get-started-with-distributed-tracing-and-grafana-tempo-using-foobar-a-demo-written-in-python/)

## Community resources

- [NodeJS](https://github.com/mnadeem/nodejs-opentelemetry-tempo)
- [Java Spring Boot](https://github.com/mnadeem/boot-opentelemetry-tempo)
- [Python](https://github.com/dgzlopes/foobar-demo)
