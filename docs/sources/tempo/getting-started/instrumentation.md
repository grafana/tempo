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
Most of these guides include complete end-to-end examples with Grafana, Loki, and Tempo.

### Instrumentation frameworks

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

## OpenTelemetry

- [OpenTelemetry Language Specific Instrumentation](https://opentelemetry.io/docs/instrumentation/)

## Jaeger
- [Jaeger Language Specific Instrumentation](https://www.jaegertracing.io/docs/latest/client-libraries/)

## Zipkin
- [Zipkin Language Specific Instrumentation](https://zipkin.io/pages/tracers_instrumentation.html)

## Grafana Blog

- [Java Spring Boot Auto-Instrumentation](/blog/2021/02/03/auto-instrumenting-a-java-spring-boot-application-for-traces-and-logs-using-opentelemetry-and-grafana-tempo/)
- [Go + OpenMetrics Exemplars](/blog/2020/11/09/trace-discovery-in-grafana-tempo-using-prometheus-exemplars-loki-2.0-queries-and-more/)
- [.NET](/blog/2021/02/11/instrumenting-a-.net-web-api-using-opentelemetry-tempo-and-grafana-cloud/)
- [Python](/blog/2021/05/04/get-started-with-distributed-tracing-and-grafana-tempo-using-foobar-a-demo-written-in-python/)

## Community Resources

- [NodeJS](https://github.com/mnadeem/nodejs-opentelemetry-tempo)
- [Java Spring Boot](https://github.com/mnadeem/boot-opentelemetry-tempo)
- [Python](https://github.com/dgzlopes/foobar-demo)
