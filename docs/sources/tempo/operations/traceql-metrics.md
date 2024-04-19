---
aliases: []
title: TraceQL metrics
menuTitle: TraceQL metrics
description: Learn about using TraceQL metrics.
weight: 550
keywords:
  - Prometheus
  - TraceQL
  - TraceQL metrics
---

# TraceQL metrics

{{< docs/experimental product="TraceQL metrics" >}}

Tempo 2.4 introduces the addition of metrics queries to the TraceQL language as an experimental feature.
Metric queries extend trace queries by applying a function to trace query results.
This powerful feature creates metrics from traces, much in the same way that LogQL metric queries create metrics from logs.
Initially, only `count_over_time` and `rate` are supported.

For example:
```
{ resource.service.name = "foo" && status = error } | rate()
```

In this case, we are calculating the rate of the erroring spans coming from the service `foo`. Rate is a `spans/sec` quantity.
Combined with the `by()` operator, this can be even more powerful!

```
{ resource.service.name = "foo" && status = error } | rate() by (span.http.route)
```

Now, we are still rating the erroring spans in the service `foo` but the metrics have been broken
down by HTTP endpoint. This might let you determine that `/api/sad` had a higher rate of erroring
spans than `/api/happy`, for example.

## Enable and use TraceQL metrics

You can use the TraceQL metrics in Grafana with any existing or new Tempo data source.
This capability is available in Grafana Cloud and Grafana (10.4 and newer).

![Metrics visualization in Grafana](/media/docs/tempo/metrics-explore-sample-2.4.png)

### Before you begin

To use the metrics generated from traces, you need to:

* Set the `local-blocks` processor to active in your `metrics-generator` configuration
* Configure a Tempo data source configured in Grafana or Grafana Cloud
* Access Grafana Cloud or Grafana 10.4

### Configure the `local-blocks` processor

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

### Evaluate query timeouts

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