---
aliases: []
title: Configure TraceQL metrics
menuTitle: Configure TraceQL metrics
description: Learn about configuring TraceQL metrics.
weight: 550
keywords:
  - Prometheus
  - TraceQL
  - TraceQL metrics
---

# Configure TraceQL metrics

{{< docs/experimental product="TraceQL metrics" >}}

TraceQL language provides metrics queries as an experimental feature.
Metric queries extend trace queries by applying a function to trace query results.
This powerful feature creates metrics from traces, much in the same way that LogQL metric queries create metrics from logs.

For more information about available queries, refer to [TraceQL metrics queries]({{< relref "../traceql/metrics-queries/" >}}).

## Before you begin

To use the metrics generated from traces, you need to:

* Set the `local-blocks` processor to active in your `metrics-generator` configuration
* Configure a Tempo data source configured in Grafana or Grafana Cloud
* Access Grafana Cloud or Grafana 10.4

## Configure the `local-blocks` processor

Once the `local-blocks` processor is enabled in your `metrics-generator`
configuration, you can configure it using the following block to make sure
it records all spans for TraceQL metrics.

Here is an example configuration:

```yaml
 metrics_generator:
  processor:
    local_blocks:
      filter_server_spans: false
  storage:
    path: /var/tempo/generator/wal
  traces_storage:
    path: /var/tempo/generator/traces
```

Refer to the [metrics-generator configuration]({{< relref "../configuration#metrics-generator" >}}) documentation for more information.

## Evaluate query timeouts

Because of their expensive nature, these queries can take a long time to run in different systems.
As such, consider increasing the timeouts in various places of
the system to allow enough time for the data to be returned.

Consider these areas when raising timeouts:

- Any proxy in front of Grafana
- Grafana data source for Prometheus pointing at Tempo
- Tempo configuration
  - `querier.search.query_timeout`
  - `server.http_server_read_timeout`
  - `server.http_server_write_timeout`

Additionally, a new `query_frontend.metrics` config has been added. The config
here will depend on the environment.

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