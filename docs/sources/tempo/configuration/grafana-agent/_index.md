---
title: Grafana Agent
weight: 600
aliases:
- /docs/tempo/grafana-agent
---

# Grafana Agent

The [Grafana Agent](https://github.com/grafana/agent) is a telemetry
collector for sending metrics, logs, and trace data to the opinionated
Grafana observability stack.

It is commonly used as a tracing pipeline, offloading traces from the
application and forwarding them to a storage backend.
The Grafana Agent tracing stack is built using OpenTelemetry.

The Grafana Agent supports receiving traces in multiple formats:
OTLP (OpenTelemetry), Jaeger, Zipkin, and OpenCensus.

On top of receiving and exporting traces, the Grafana Agent contains many
features that make your distributed tracing system more robust, and
leverages all the data that is processed in the pipeline.

## Architecture

The Grafana Agent can be configured to run a set of tracing pipelines to collect data from your applications and write it to Tempo.
Pipelines are built using OpenTelemetry,
and consist of `receivers`, `processors` and `exporters`. The architecture mirrors that of the OTel Collector's [design](https://github.com/open-telemetry/opentelemetry-collector/blob/846b971758c92b833a9efaf742ec5b3e2fbd0c89/docs/design.md).
See the [configuration reference](/docs/agent/latest/configuration/traces-config) for all available configuration options.
For a quick start, refer to this [blog post](/blog/2020/11/17/tracing-with-the-grafana-cloud-agent-and-grafana-tempo/).

<p align="center"><img src="https://raw.githubusercontent.com/open-telemetry/opentelemetry-collector/846b971758c92b833a9efaf742ec5b3e2fbd0c89/docs/images/design-pipelines.png" alt="Tracing pipeline architecture"></p>

This allows you to configure multiple distinct tracing
pipelines, each of which collects separate spans and sends them to different
backends.

### Receiving traces

The Grafana Agent supports multiple ingestion receivers:
OTLP (OpenTelemetry), Jaeger, Zipkin, OpenCensus and Kafka.

Each tracing pipeline can be configured to receive traces in all these formats.
Traces that arrive to a pipeline will go through the receivers/processors/exporters defined in it.

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
For example, for Kubernetes users this means that you can dynamically attach metadata for namespace, pod, and name of the container sending spans.

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
This means you can use the same scrape_configs between your metrics, logs, and traces to get the same set of labels,
and easily transition between your observability data when moving from your metrics, logs, and traces.

To configure it, refer to the `scrape_configs` block in the [configuration reference](/docs/agent/latest/configuration/traces-config).

#### Trace discovery through automatic logging

Automatic logging writes well formatted log lines to help with trace discovery.

For a closer look into the feature, visit [Automatic logging]({{< relref "./automatic-logging" >}}).

#### Tail-based sampling

The Agent implements tail-based sampling for distributed tracing systems and multi-instance Agent deployments.
With this feature, sampling decisions can be made based on data from a trace, rather than exclusively with probabilistic methods.

For a detailed description, go to [Tail-based sampling]({{< relref "./tail-based-sampling" >}}).

#### Generating metrics from spans

The Agent can take advantage of the span data flowing through the pipeline to generate Prometheus metrics.

Go to [Span metrics]({{< relref "./span-metrics" >}}) for a more detailed explanation of the feature.

#### Service graph metrics

Service graph metrics represent the relationships between services within a distributed system.

This service graphs processor builds a map of services by analyzing traces, with the objective to find _edges_.
Edges are spans with a parent-child relationship, that represent a jump (e.g. a request) between two services.
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
