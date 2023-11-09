---
title: Tempo documentation
description: Grafana Tempo is an open source distributed tracing backend.
aliases:
  - /docs/tempo/
cascade:
  GRAFANA_VERSION: next
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
