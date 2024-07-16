---
title: Grafana Agent
description: Configure the Grafana Agent to work with Tempo
weight: 600
aliases:
- /docs/tempo/grafana-agent
---

# Grafana Agent

{{< docs/shared source="alloy" lookup="agent-deprecation.md" version="next" >}}

[Grafana Agent](https://github.com/grafana/agent) is a telemetry
collector for sending metrics, logs, and trace data to the opinionated
Grafana observability stack.

{{< admonition type="note">}} 
Grafana Alloy provides tooling to convert your Agent Static or Flow configuration files into a format that can be used by Alloy.

For more information, refer to [Migrate to Alloy](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/grafana-alloy/migrate-alloy).
{{< /admonition>}}

It's commonly used as a tracing pipeline, offloading traces from the
application and forwarding them to a storage backend.
Grafana Agent tracing stack is built using OpenTelemetry.

Grafana Agent supports receiving traces in multiple formats:
OTLP (OpenTelemetry), Jaeger, Zipkin, and OpenCensus.

On top of receiving and exporting traces, Grafana Agent contains many
features that make your distributed tracing system more robust, and
leverages all the data that's processed in the pipeline.

## Agent modes

Grafana Agent is available in two different variants:

* [Static mode](/docs/agent/latest/static): The original Grafana Agent.
* [Flow mode](/docs/agent/latest/flow): The new, component-based Grafana Agent.

Grafana Agent Flow configuration files are [written in River](/docs/agent/latest/flow/concepts/config-language/).
Static configuration files are [written in YAML](/docs/agent/latest/static/configuration/).
Examples in this document are for Flow mode.

For more information, refer to the [Introduction to Grafana Agent](/docs/agent/latest/about/).

## Architecture

The Grafana Agent can be configured to run a set of tracing pipelines to collect data from your applications and write it to Tempo.
Pipelines are built using OpenTelemetry,
and consist of `receivers`, `processors`, and `exporters`.
The architecture mirrors that of the OTel Collector's [design](https://github.com/open-telemetry/opentelemetry-collector/blob/846b971758c92b833a9efaf742ec5b3e2fbd0c89/docs/design.md).
See the [configuration reference](/agent/latest/static/configuration/traces-config/) for all available configuration options.

<p align="center"><img src="https://raw.githubusercontent.com/open-telemetry/opentelemetry-collector/846b971758c92b833a9efaf742ec5b3e2fbd0c89/docs/images/design-pipelines.png" alt="Tracing pipeline architecture"></p>

This allows you to configure multiple distinct tracing
pipelines, each of which collects separate spans and sends them to different
backends.

### Receiving traces
<!-- vale Grafana.Parentheses = NO -->
The Grafana Agent supports multiple ingestion receivers:
OTLP (OpenTelemetry), Jaeger, Zipkin, OpenCensus, and Kafka.
<!-- vale Grafana.Parentheses = YES -->

Each tracing pipeline can be configured to receive traces in all these formats.
Traces that arrive to a pipeline go through the receivers/processors/exporters defined in that pipeline.

### Pipeline processing

The Grafana Agent processes tracing data as it flows through the pipeline to make the distributed tracing system more reliable and leverage the data for other purposes such as trace discovery, tail-based sampling, and generating metrics.

#### Batching

The Agent supports batching of traces.
Batching helps better compress the data, reduces the number of outgoing connections, and is a recommended best practice.
To configure it, refer to the `batch` block in the [configuration reference](/docs/agent/latest/configuration/traces-config).

#### Attributes manipulation

The Grafana Agent allows for general manipulation of attributes on spans that pass through this agent.
A common use may be to add an environment or cluster variable.
To configure it, refer to the `attributes` block in the [configuration reference](/docs/agent/latest/configuration/traces-config).

#### Attaching metadata with Prometheus Service Discovery

Prometheus Service Discovery mechanisms enable you to attach the same metadata to your traces as your metrics.
For example, for Kubernetes users this means that you can dynamically attach metadata for namespace, Pod, and name of the container sending spans.

```yaml
traces:
  ...
  scrape_configs:
  - bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
    job_name: kubernetes-pods
    kubernetes_sd_configs:
    - role: pod
    relabel_configs:
    - source_labels: [__meta_kubernetes_namespace]
      target_label: namespace
    - source_labels: [__meta_kubernetes_pod_name]
      target_label: pod
    - source_labels: [__meta_kubernetes_pod_container_name]
      target_label: container
    tls_config:
      ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
      insecure_skip_verify: false
```

This feature isn’t just useful for Kubernetes users, however.
All of Prometheus' [various service discovery mechanisms](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#configuration-file) are supported here.
This means you can use the same `scrape_configs` between your metrics, logs, and traces to get the same set of labels,
and easily transition between your observability data when moving from your metrics, logs, and traces.

Refer to the `scrape_configs` block in the [configuration reference](/docs/agent/latest/configuration/traces-config).

#### Trace discovery through automatic logging

Automatic logging writes well formatted log lines to help with trace discovery.

For a closer look into the feature, visit [Automatic logging]({{< relref "./automatic-logging" >}}).

#### Tail-based sampling

The Agent implements tail-based sampling for distributed tracing systems and multi-instance Agent deployments.
With this feature, sampling decisions can be made based on data from a trace, rather than exclusively with probabilistic methods.

For a detailed description, go to [Tail-based sampling]({{< relref "./tail-based-sampling" >}}).

For additional information, refer to the blog post, [An introduction to trace sampling with Grafana Tempo and Grafana Agent](/blog/2022/05/11/an-introduction-to-trace-sampling-with-grafana-tempo-and-grafana-agent).

#### Generating metrics from spans

The Agent can take advantage of the span data flowing through the pipeline to generate Prometheus metrics.

Go to [Span metrics]({{< relref "./span-metrics" >}}) for a more detailed explanation of the feature.

#### Service graph metrics

Service graph metrics represent the relationships between services within a distributed system.

This service graphs processor builds a map of services by analyzing traces, with the objective to find _edges_.
Edges are spans with a parent-child relationship, that represent a jump, such as a request, between two services.
The amount of requests and their duration are recorded as metrics, which are used to represent the graph.

To read more about this processor, go to its [section]({{< relref "./service-graphs" >}}).

### Exporting spans

The Grafana Agent can export traces to multiple different backends for every tracing pipeline.
Exporting is built using OpenTelemetry Collector's [OTLP exporter](https://github.com/open-telemetry/opentelemetry-collector/blob/846b971758c92b833a9efaf742ec5b3e2fbd0c89/exporter/otlpexporter/README.md).
The Agent supports exporting tracing in OTLP format.

Aside from endpoint and authentication, the exporter also provides mechanisms for retrying on failure,
and implements a queue buffering mechanism for transient failures, such as networking issues.

To see all available options,
refer to the `remote_write` block in the [Agent configuration reference](/docs/agent/latest/configuration/traces-config).
