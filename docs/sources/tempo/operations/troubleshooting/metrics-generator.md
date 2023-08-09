---
title: Troubleshoot metrics-generator
menuTitle: Metrics-generator
description: Gain an understanding of how to debug metrics quality issues.
weight: 500
---

# Troubleshoot metrics-generator

If you are concerned with data quality issues in the metrics-generator, we'd first recommend:

- Reviewing your telemetry pipeline to determine the number of dropped spans. We are only looking for major issues here.
- Reviewing the [service graph documentation]({{< relref "../../metrics-generator/service_graphs" >}}) to understand how they are built.

If everything seems ok from these two perspectives, consider the following topics to help resolve general issues with all metrics and span metrics specifically.

## All metrics

### Dropped spans in the distributor

The distributor has a queue of outgoing spans to the metrics-generators. If that queue is full then the distributor
will drop spans before they reach the generator. Use the following metric to determine if that is happening:

```
sum(rate(tempo_distributor_queue_pushes_failures_total{}[1m]))
```

### Failed pushes to the generator

For any number of reasons, the distributor can fail a push to the generators. Use the following metric to
determine if that is happening:

```
sum(rate(tempo_distributor_metrics_generator_pushes_failures_total{}[1m]))
```

### Discarded spans in the generator

Spans are rejected from being considered by the metrics-generator by a configurable slack time as well as due to user
configurable filters. You can see the number of spans rejected by reason using this metric:

```
sum(rate(tempo_metrics_generator_spans_discarded_total{}[1m])) by (reason)
```

If you are dropping a lot of spans due to your filters, you will need to adjust them. If you are dropping a lot of spans
due to the ingestion slack time, consider adjusting this setting:

```
metrics_generator:
  metrics_ingestion_time_range_slack: 30s
```

### Max active series

The generator protects itself and your remote write target by having a maximum number of series it will produce. To 
determine if series are being dropped due to this limit check:

```
sum(rate(tempo_metrics_generator_registry_series_limited_total{}[1m]))
```

Use the following setting to update the limit:

```
overrides:
  metrics_generator_max_active_series: 0
```

### Remote write failures

For any number of reasons the generator may fail a write to the remote write target. Use the following metrics to
determine if that is happening:

```
sum(rate(prometheus_remote_storage_samples_failed_total{}[1m]))
sum(rate(prometheus_remote_storage_samples_dropped_total{}[1m]))
```

## Service graph metrics

Service graphs have additional configuration which can impact the quality of the output metrics. 

### Expired edges

The following metrics can be used to determine how many edges are failing to find a match.

Rate of edges that have expired without a match:
```
sum(rate(tempo_metrics_generator_processor_service_graphs_expired_edges{}[1m]))
```

Rate of all edges:
```
sum(rate(tempo_metrics_generator_processor_service_graphs_edges{}[1m]))
```

If you are seeing a large number of edges expire without a match, consider adjusting the `wait` setting. This
controls how long the metrics generator waits to find a match before it gives up.

```
metrics_generator:
  processor:
    service_graphs:
      wait: 10s
```

### Service graph max items

In order to limit the total amount of memory that the service graph processor uses there is a maximum number
of edges it will track at once. To determine if edges are being dropped due to this limit check:

```
sum(rate(tempo_metrics_generator_processor_service_graphs_dropped_spans{}[1m]))
```

To adjust the maximum amount of edges it will track use:

```
metrics_generator:
  processor:
    service_graphs:
      max_items: 10000
```
