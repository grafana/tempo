---
title: Troubleshoot Metrics Generator
menuTitle: Metrics Generator
description: Gain an understanding of how to debug metrics quality issues.
weight: 500
---

If you are concerned with data quality issues in the metrics generator, we'd first recommend:

- Reviewing your telemetry pipeline to determine the number of dropped spans. We are only looking for major issues here.
- Reviewing the [service graph documentation]({{< relref "../../metrics-generator/service_graphs" >}}) to understand how they are built.

If everything seems ok from these two perspectives consider the following.

## All metrics

Spans are rejected from being considered by the metrics generator by a configurable slack time as well as due to user
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

## Service graphs

Service graphs have additional configuration which can impact the quality of the output metrics. The following metrics
can be used to determine how many edges are failing to find a match.

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