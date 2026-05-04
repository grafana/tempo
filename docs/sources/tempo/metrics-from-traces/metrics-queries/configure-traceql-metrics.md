---
aliases:
  - ../../operations/traceql-metrics/ # /docs/tempo/next/operations/traceql-metrics/
title: Configure TraceQL metrics
menuTitle: Configure TraceQL metrics
description: Learn about configuring TraceQL metrics.
weight: 400
keywords:
  - Prometheus
  - TraceQL
  - TraceQL metrics
---

# Configure TraceQL metrics

TraceQL language provides metrics queries as a feature.
Metric queries extend trace queries by applying a function to trace query results.
This powerful feature creates metrics from traces, much in the same way that LogQL metric queries create metrics from logs.

## Before you begin

To use the metrics generated from traces, you need to:

- Configure a Tempo data source in Grafana or Grafana Cloud ([documentation](/docs/grafana/<GRAFANA_VERSION>/datasources/tempo/configure-tempo-data-source/))
- Access Grafana Cloud or Grafana version 10.4 or later

## Evaluate query timeouts

Because of their expensive nature, these queries can take a long time to run.
As such, consider increasing the timeouts in various places of
the system to allow enough time for the data to be returned.

Consider these areas when raising timeouts:

- Any proxy in front of Grafana
- Grafana data source for Prometheus pointing at Tempo
- Tempo configuration
  - `querier.search.query_timeout`
  - `server.http_server_read_timeout`
  - `server.http_server_write_timeout`

## Set TraceQL metrics query options

The `query_frontend.metrics` configuration block controls all TraceQL metrics queries.
The configuration depends on the environment.

{{< admonition type="note" >}}
The default maximum time range for a metrics query is 24 hours, configured using the `query_frontend.metrics.max_duration` parameter.

This is different to the default TraceQL maximum time range of 168 hours (7 days).

{{< /admonition >}}

The `query_frontend.metrics.query_backend_after` parameter controls the boundary between querying the live-store and backend storage.
Time ranges older than `query_backend_after` (default `15m`) are searched in backend/object storage only, while more recent data is queried from the live-store.

For example, in a cloud environment, smaller jobs with more concurrency may be
desired due to the nature of scale on the backend.

```yaml
query_frontend:
  metrics:
    concurrent_jobs: 1000
    target_bytes_per_job: 2.25e+08 # ~225MB
    interval: 30m0s
```

For an on-prem backend, you can improve query times by lowering the concurrency,
while increasing the job size.

```yaml
query_frontend:
  metrics:
    concurrent_jobs: 8
    target_bytes_per_job: 1.25e+09 # ~1.25GB
```

## Sampling and performance optimization

TraceQL metrics queries support sampling hints to improve performance on large datasets. Refer to the [TraceQL metrics sampling](/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/metrics-queries/sampling-guide/) documentation for more information.

When using sampling in your TraceQL metrics queries, consider:

- **Timeout settings:** Sampled queries run faster but may still benefit from adequate timeouts
- **Concurrent jobs:** Sampling reduces per-job processing time, allowing higher concurrency
