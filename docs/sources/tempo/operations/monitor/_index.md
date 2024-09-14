---
title: Monitor Tempo
menuTitle: Monitor Tempo
description: Use polling, alerts, and dashboards to monitor Tempo in production.
weight: 20
aliases:
- ./monitoring ## https://grafana.com/docs/tempo/latest/operations/monitoring/
---

# Monitor Tempo

Tempo is instrumented to expose metrics, logs, and traces.
Furthermore, the Tempo repository has a [mixin](https://github.com/grafana/tempo/tree/main/operations/tempo-mixin) that includes a
set of dashboards, rules, and alerts.
Together, these can be used to monitor Tempo in production.

## Instrumentation

Metrics, logs, and traces from Tempo can be collected to observe its services and functions.

### Metrics

Tempo is instrumented with [Prometheus metrics](https://prometheus.io/) and emits RED metrics for most services and backends.
RED metrics are a standardized format for monitoring microservices, where R stands for requests, E stands for errors, and D stands for duration.

The [Tempo mixin](#dashboards) provides several dashboards using these metrics.

### Logs

Tempo emits logs in the `key=value` ([logfmt](https://brandur.org/logfmt)) format.

### Traces

Tempo uses the [OpenTelemetry SDK](https://github.com/open-telemetry/opentelemetry-go) for tracing instrumentation.
The complete read path and some parts of the write path of Tempo are instrumented for tracing.

You can configure the tracer [using environment variables](https://opentelemetry.io/docs/languages/sdk-configuration/otlp-exporter/).
To enable tracing, set one of the following: `OTEL_EXPORTER_OTLP_ENDPOINT` or `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT`.

The OpenTelemetry SDK uses OTLP/HTTP by default, which can be configured with `OTEL_EXPORTER_OTLP_PROTOCOL`.

## Polling

Tempo maintains knowledge of the state of the backend by polling it on regular intervals.
There are currently only two components that need this knowledge and, consequently, only two that poll the backend: compactors and queriers.

Refer to [Use polling to monitor the backend status]({{< relref "./polling" >}}) for Tempo.

## Dashboards

The [Tempo mixin](https://github.com/grafana/tempo/tree/main/operations/tempo-mixin) has four Grafana dashboards in the `yamls` folder that you can download and import into your Grafana UI.
These dashboards work well when you run Tempo in a Kubernetes (k8s) environment and metrics scraped have the
`cluster` and `namespace` labels.

### Tempo Reads dashboard

> This is available as `tempo-reads.json`.

The Reads dashboard gives information on Requests, Errors, and Duration (RED) on the query path of Tempo.
Each query touches the Gateway, Tempo-Query, Query-Frontend, Queriers, Ingesters, the backend, and Cache, if present.

Use this dashboard to monitor the performance of each of the mentioned components and to decide the number of
replicas in each deployment.

### Tempo Writes dashboard

> This is available as `tempo-writes.json`.

The Writes dashboard gives information on RED on the write/ingest path of Tempo.
A write query touches the Gateway, Distributors, Ingesters, and the backend.
This dashboard also gives information
on the number of operations performed by the Compactor to the backend.

Use this dashboard to monitor the performance of each of the mentioned components and to decide the number of
replicas in each deployment.

### Tempo Resources dashboard

> This is available as `tempo-resources.json`.

The Resources dashboard provides information on `CPU`, `Container Memory`, and `Go Heap Inuse`.
This dashboard is useful for resource provisioning for the different Tempo components.

Use this dashboard to see if any components are running close to their assigned limits.

### Tempo Operational dashboard

> This is available as `tempo-operational.json`.

The Tempo Operational dashboard deserves special mention because it is probably a stack of dashboard anti-patterns.
It's big and complex, doesn't use `jsonnet`, and displays far too many metrics in one place.
For just getting started, the RED dashboards are great places to learn how to monitor Tempo in an opaque way.

This dashboard is included in the Tempo repository for two reasons:

- The dashboard provides a stack of metrics for other operators to consider monitoring while running Tempo.
- We want the dashboard in our internal infrastructure and we vendor the `tempo-mixin` to do this.

## Rules and alerts

The Rules and Alerts are available as [YAML files in the compiled mixin](https://github.com/grafana/tempo/tree/main/operations/tempo-mixin-compiled) on the repository.

To set up alerting, download the provided JSON files and configure them for use on your Prometheus monitoring server.

Check the [runbook](https://github.com/grafana/tempo/blob/main/operations/tempo-mixin/runbook.md) to understand the
various steps that can be taken to fix firing alerts.
