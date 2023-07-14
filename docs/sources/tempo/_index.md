---
title: Tempo documentation
description: Grafana Tempo is an open source distributed tracing backend.
aliases:
  - /docs/tempo/
cascade:
  glossary:
    active series: A time series that receives new data points or samples.
    cardinality: The total combination of key/value pairs, such as labels and label values for a given metric series or log stream, and how many unique combinations they generate.
    data source: A basic storage for data such as a database, a flat file, or even live references or measurements from a device. A file, database, or service that provides data. For example, traces data is imported into Grafana by configuring and enabling a Tempo data source.
    exemplar: Any data that serves as a detailed example of one of the observations aggregated into a metric. An exemplar contains the observed value together with an optional timestamp and arbitrary trace IDs, which are typically used to reference a trace.
    log: Chronological events, usually text-based, allowing for the diagnosis of problems. Logs can provide informational context, such as detailed records of all events during user interactions, for example, when events happen, who used the system, status messages, etc.
    metric: A number that helps an operator understand the state of a system, such as the number of active users, error count, average response time, and more.
    span: A unit of work done within a trace.
    trace: An observed execution path of a request through a distributed system.
---

# Tempo documentation

<p align="center"><img src="logo_and_name.png" alt="Tempo Logo"></p>

Grafana Tempo is an open source, easy-to-use, and high-volume distributed tracing backend. Tempo is cost-efficient, and only requires an object storage to operate. Tempo is deeply integrated with Grafana, Mimir, Prometheus, and Loki. You can use Tempo with open-source tracing protocols, including Jaeger, Zipkin, or OpenTelemetry.

Tempo integrates well with a number of existing open source tools:

- **Grafana** ships with native support for Tempo using the built-in [Tempo data source](/docs/grafana/latest/datasources/tempo/).
- **Grafana Loki**, with its powerful query language [LogQL v2](/blog/2020/10/28/loki-2.0-released-transform-logs-as-youre-querying-them-and-set-up-alerts-within-loki/) allows you to filter requests that you care about, and jump to traces using the [Derived fields support in Grafana](/docs/grafana/latest/datasources/loki/#derived-fields).
- **Prometheus exemplars** let you jump from Prometheus metrics to Tempo traces by clicking on recorded exemplars. Read more about this integration in the blog post [Intro to exemplars, which enable Grafana Tempo’s distributed tracing at massive scale](/blog/2021/03/31/intro-to-exemplars-which-enable-grafana-tempos-distributed-tracing-at-massive-scale/).

<p align="center"><img src="getting-started/assets/trace_custom_metrics_dash.png" alt="Trace visualization in Grafana "></p>

Grafana Tempo builds an index from the high-cardinality trace-id field. Because Tempo uses an object store as a backend, Tempo can query many blocks simultaneously, so queries are highly parallelized.
For more information, see [Architecture]({{< relref "./operations/architecture" >}}).

## Learn more about Tempo

{{< section >}}
