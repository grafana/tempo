---
title: Monitor Tempo
menuTitle: Monitor Tempo
description: Use polling, alerts, and dashboards to monitor Tempo in production.
weight: 200
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

#### Query frontend inspected bytes

Use `tempo_query_frontend_bytes_inspected_total` to monitor how many bytes the query frontend reads from disk and object storage.
This counter is emitted per `tenant` and `op` (`traces`, `search`, `metadata`, `metrics`).
Because cached responses from queriers are excluded, it reflects actual storage and network I/O.

For PromQL examples and alerting guidance, refer to [Query query IO and time stamp distance](/docs/tempo/<TEMPO_VERSION>/operations/monitor/query-io-and-timestamp-distance/).

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
Several components need this knowledge, including schedulers, workers, queriers, and query-frontends.

Refer to [Use polling to monitor the backend status](polling/) for Tempo.

## Dashboards

The [Tempo mixin](https://github.com/grafana/tempo/tree/main/operations/tempo-mixin) has eight Grafana dashboards in the `dashboards` folder that you can download and import into your Grafana UI.
These dashboards work well when you run Tempo in a Kubernetes (k8s) environment and metrics scraped have the
`cluster` and `namespace` labels.

### Tempo Reads dashboard

> This is available as `tempo-reads.json`.

The Reads dashboard gives information on Requests, Errors, and Duration (RED) on the query path of Tempo.
Each query touches the Gateway, Query-Frontend, Queriers, Live-stores, Memcached, and the backend.

Use this dashboard to monitor the performance of each of the mentioned components and to decide the number of
replicas in each deployment.

### Tempo Writes dashboard

> This is available as `tempo-writes.json`.

The Writes dashboard gives information on RED on the write/ingest path of Tempo.
A write query touches the Gateway, Distributors, and Kafka.
The dashboard also shows compaction activity against Memcached and the backend.

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

### Tempo Backend Work dashboard

> This is available as `tempo-backendwork.json`.

The Backend Work dashboard monitors blocklist maintenance, compaction jobs, and backend component resources.
It tracks blocklist length and poll duration, active and completed compaction jobs, failure and retry rates, and objects written and combined during compaction.
The dashboard also shows CPU and memory usage for the backend-scheduler and backend-worker components.

Use this dashboard to monitor compaction health, detect stalled or failing jobs, and right-size backend-scheduler and backend-worker resources.

### Tempo Block Builder dashboard

> This is available as `tempo-block-builder.json`.

The Block Builder dashboard monitors the Kafka-based ingest path introduced.
It tracks Kafka fetch rates and read throughput, flushed blocks per second, and per-partition lag in both records and seconds.
Heatmaps show partition section and cycle durations, and resource panels track CPU and memory usage.

Use this dashboard to monitor block-builder throughput, detect partition lag, and ensure the ingest pipeline keeps up with incoming trace data.

### Tempo Rollout Progress dashboard

> This is available as `tempo-rollout-progress.json`.

The Rollout Progress dashboard tracks the health of a Tempo deployment during upgrades and rollouts.
It breaks down write and read requests by status code (2xx, 4xx, 5xx), shows 99th percentile latency, and counts unhealthy Pods.
A version panel shows the number of pods running each version, and a latency comparison panel shows current latency against the previous 24-hour baseline.

Use this dashboard during upgrades to confirm that new versions aren't introducing errors or latency regressions.

### Tempo Tenants dashboard

> This is available as `tempo-tenants.json`.

The Tenants dashboard provides per-tenant visibility into ingestion, reads, storage, and metrics generation.
It shows a limits table alongside distributor bytes and spans per second, live trace counts, query rates for ID lookups and searches, blocklist length, outstanding compactions, and metrics-generator bytes and active series.

Use this dashboard in multitenant deployments to identify tenants with high ingestion rates, query volumes, or storage growth.

## Rules and alerts

The Rules and Alerts are available as [YAML files in the compiled mixin](https://github.com/grafana/tempo/tree/main/operations/tempo-mixin-compiled) on the repository.

To set up alerting, download the provided JSON files and configure them for use on your Prometheus monitoring server.

Check the [runbook](https://github.com/grafana/tempo/blob/main/operations/tempo-mixin/runbook.md) to understand the
various steps that can be taken to fix firing alerts.
