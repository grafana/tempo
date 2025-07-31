---
title: Grafana Tempo
description: Grafana Tempo is an open source distributed tracing backend.
aliases:
  - /docs/tempo/
cascade:
  GRAFANA_VERSION: next
hero:
  title: Grafana Tempo
  level: 1
  image: /static/assets/img/blog/tempo.png
  width: 110
  height: 110
  description: >-
    Grafana Tempo is an open-source, easy-to-use, and high-scale distributed tracing backend. Tempo lets you search for traces, generate metrics from spans, and link your tracing data with logs and metrics.
cards:
  title_class: pt-0 lh-1
  items:
    - title: Learn about tracing
      href: ./introduction/
      description: What is distributed tracing? Learn about traces and how you can use them, how you can instrument your app for tracing, and how you can visualize tracing data in Grafana.
    - title: Solutions and use cases
      href: ./solutions-with-traces/
      description: Learn how tracing data can help you understand application insights and performance as well as help triage issues in your services and applications.
    - title: Set up for tracing
      href: ./set-up-for-tracing/
      description: Plan your deployment to meet your needs, deploy Tempo, test your installation, and instrument your app or services for traces.
    - title: Manage Tempo
      href: ./operations/
      description: Learn about Tempo architecture, best practices, Parquet backend, dedicated attribute columns, and more.
    - title: Metrics and tracing
      href: ./metrics-from-traces/
      description: Learn about metrics created from traces, including the metrics-generator processor and TraceQL metrics.
    - title: Query with TraceQL
      href: ./traceql/
      description: Inspired by PromQL and LogQL, TraceQL is a query language designed for selecting traces in Tempo. This query language lets you precisely and easily select spans and jump directly to the spans fulfilling the specified conditions.
---

{{< docs/hero-simple key="hero" >}}

---

## Overview

Distributed tracing visualizes the lifecycle of a request as it passes through a set of applications.

Tempo is cost-efficient and only requires an object storage to operate.
Tempo is deeply integrated with Grafana, Mimir, Prometheus, and Loki.
You can use Tempo with open source tracing protocols, including Jaeger, Zipkin, or OpenTelemetry.

{{< figure src="/media/docs/tempo/intro/trace_custom_metrics_dash.png" alt="Trace visualization in Grafana" class="w-100p" link-class="w-fit mx-auto d-flex flex-direction-column" >}}

Tempo integrates well with a number of open source tools:

- **Grafana** ships with native support using the built-in [Tempo data source](/docs/grafana/<GRAFANA_VERSION>/datasources/tempo/).
- **Grafana Loki**, with its powerful query language LogQL v2 lets you filter requests that you care about, and jump to traces using the [Derived fields support in Grafana](/docs/grafana/<GRAFANA_VERSION>/datasources/loki/#derived-fields).
- **Prometheus exemplars** let you jump from Prometheus metrics to Tempo traces by clicking on recorded exemplars.

## Explore

{{< card-grid key="cards" type="simple" >}}
