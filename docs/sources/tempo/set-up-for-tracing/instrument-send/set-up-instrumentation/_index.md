---
title: Set up instrumentation
menuTitle: Set up instrumentation
description: Learn how to set up instrumentation for distributed tracing.
weight: 500
---

# Set up instrumentation

Client instrumentation is the first building block to a functioning distributed tracing visualization pipeline.
Client instrumentation is the process of adding instrumentation points in the application that create and offload spans.

When sending traces to Tempo, you can choose between four methods:
* Auto-instrumentation applies instrumentation automatically using agents or middleware, without code changes.
* Zero-code instrumentation, which uses eBPF technology to instrument applications without code changes.
* Manual instrumentation involves adding code to create spans and traces, giving full control over collected data.
* Hybrid instrumentation, which combines auto and manual instrumentation, using automatic for most code and manual for custom tracing logic.



In generate and gather traces, you need to:

1. Set up a collector to receive traces from your application
1. Select an instrumentation method to use with your application
1. Instrument your application to generate traces

## How instrumentation works

To add instrumentation, the code for a service uses a Software Development Kit (SDK) which supplies language-specific libraries that allow the:

* Creation of a new trace, starting with a new root span.
* Addition of new spans, that are siblings of children of pre-existing spans.
* Addition of span attributes to add contextual information to each span, as well as span links and events.
* Closure of spans when a unit of work is complete.

Adding code to carry out these operations is known as Manual Instrumentation, as it requires manual intervention by an engineer to write code to deal with traces, as well as to determine where in the code traces/spans should start, the attributes and other data that should be attached to spans, and where traces/spans should end.

There is an alternative/companion to manual instrumentation, Auto-instrumentation. Auto-instrumentation is the act of allowing a tracing SDK to determine where traces/spans should start, what information should be added to spans, and where traces/spans should stop. Essentially, manual instrumentation is pre-packaged for a large number of popular frameworks and libraries which are used inside a service's code, and it is these libraries/frameworks that are actually emitting spans for a trace.

These libraries usually include those dealing with networking, so for example a request coming into a service might be via an auto-instrumented HTTP library, which would then start a trace until it sent a response back via HTTP to the requester. Along the course of the request, the service might use other libraries that process data, and if they are also auto-instrumented then new spans will be generated for the trace that include suitable attributes.


## Collect and forward traces with auto-instrumentation using Grafana Alloy or OpenTelemetry collectors

You can send data from your application using Grafana Alloy or OpenTelemetry Collector (OTel) collectors.

[Grafana Alloy](https://grafana.com/docs/alloy/latest/) is a vendor-neutral distribution of the OpenTelemetry (OTel) Collector.
Alloy uniquely combines the very best OSS observability signals in the community.
Grafana Alloy uses configuration file written using River.

Alloy is a component that runs alongside your application and periodically gathers tracing data from it.
This method is suitable when you want to collect tracing from applications without modifying their source code.

Here's how it works:

1. Install and configure the collector on the same machine or container where your application is running.
2. The collector periodically retrieves your application's performance tracing data, regardless of the language or technology stack your application is using.
3. The captured traces are then sent to the Tempo server for storage and analysis.

Using a collector provides a hassle-free option, especially when dealing with multiple applications or microservices, allowing you to centralize the profiling process without changing your application's codebase.

Refer to [Collect and forward data with Grafana Alloy](https://grafana.com/docs/alloy/<ALLOY_VERSION>/collect/) for examples of collecting data.

If you are using OTel or Alloy, refer to [Instrument an application with OpenTelemetry](https://grafana.com/docs/opentelemetry/instrument/) for more information. These instructions are specific to Grafana Cloud, but can be adapted for self-hosted Tempo.

## Use zero-code instrumentation with Grafana Beyla

Grafana Beyla is an eBPF-based application auto-instrumentation tool to easily get started with Application Observability. Beyla uses eBPF to automatically inspect application executables and the OS networking layer, and capture trace spans related to web transactions and Rate Errors Duration (RED) metrics for Linux HTTP/S and gRPC services. All data capture occurs without any modifications to application code or configuration.

Refer to [Set up Beyla](https://grafana.com/docs/beyla/<BEYLA_VERSION>/setup/) for information about how to instrument using Beyla.

## Instrumentation frameworks

Most of the popular client instrumentation frameworks have SDKs in the most commonly used programming languages.
You should pick one according to your application needs.

OpenTelemetry has the most active development in the community and may be a better long-term choice.

* [OpenTelemetry](https://opentelemetry.io/docs/concepts/instrumenting/)
* [Zipkin](https://zipkin.io/pages/tracers_instrumentation)

## OpenTelemetry

A collection of tools, APIs, and SDKs, OpenTelemetry helps engineers instrument, generate, collect, and export telemetry data such as metrics, logs, and traces, to analyze software performance and behavior.
For more information refer to [OpenTelemetry overview](https://grafana.com/oss/opentelemetry/).

If you are using OTel with Grafana Cloud, refer to [Instrument an application with OpenTelemetry](https://grafana.com/docs/opentelemetry/instrument/) for more information.


### Use OpenTelemetry auto-instrumentation frameworks

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

## Use Zipkin to auto-instrumentation

Zipkin is a distributed tracing system that helps gather timing data needed to troubleshoot latency problems in microservice architectures.

Refer to the [Zipkin Language Specific Instrumentation](https://zipkin.io/pages/tracers_instrumentation.html) documentation for more information.

If you are using Zipkin with Alloy, refer to the Zipkin receiver, [otelcol.receiver.zipkin documentation](https://grafana.com/docs/alloy/<ALlOY_VERSION>/reference/components/otelcol/otelcol.receiver.zipkin/).

In addition, you can use Zipkin to instrument a library, refer to [Instrumenting a library with Zipkin](https://zipkin.io/pages/instrumenting.html)

Within Grafana, you can also use these Zipkin specific features:
* [Zipkin data source](https://grafana.com/docs/grafana/latest/datasources/zipkin/)
* [Monitor Zipkin with Prometheus and Grafana Cloud](https://grafana.com/docs/grafana-cloud/send-data/metrics/metrics-prometheus/prometheus-config-examples/the-zipkin-community-zipkin/)
*

<!-- update these blog links
## Grafana Blog

The Grafana blog periodically features instrumentation posts.

- [How to configure OpenTelemetry .NET automatic instrumentation with Grafana Cloud](https://grafana.com/blog/2023/10/31/how-to-configure-opentelemetry-.net-automatic-instrumentation-with-grafana-cloud)
- [Java Spring Boot Auto-Instrumentation](https://grafana.com/blog/2021/02/03/auto-instrumenting-a-java-spring-boot-application-for-traces-and-logs-using-opentelemetry-and-grafana-tempo/)
- [Go + OpenMetrics Exemplars](https://grafana.com/blog/2020/11/09/trace-discovery-in-grafana-tempo-using-prometheus-exemplars-loki-2.0-queries-and-more/)
- [.NET](https://grafana.com/blog/2021/02/11/instrumenting-a-.net-web-api-using-opentelemetry-tempo-and-grafana-cloud/)
- [Python](https:/grafana.com/blog/2021/05/04/get-started-with-distributed-tracing-and-grafana-tempo-using-foobar-a-demo-written-in-python/)
-->
## Community resources

- [NodeJS](https://github.com/mnadeem/nodejs-opentelemetry-tempo)
- [Java Spring Boot](https://github.com/mnadeem/boot-opentelemetry-tempo)
- [Python](https://github.com/dgzlopes/foobar-demo)
