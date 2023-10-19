---
aliases:
- /docs/tempo/v1.2.1/grafana-agent/span-metrics/
title: Generating metrics from spans
---

# Generating metrics from spans

Span metrics allow you to generate metrics from your tracing data automatically.
Span metrics aggregates request, error and duration (RED) metrics from span data.
Metrics are exported in Prometheus format.

There are two options available for exporting metrics: using remote write to a Prometheus compatible backend or serving the metrics locally and scraping them.

<p align="center"><img src="../span-metrics.png" alt="Span metrics overview"></p>

Span metrics generate two metrics: a counter that computes requests, and a histogram that computes operation’s durations.

Span metrics are of particular interest if your system is not monitored with metrics,
but it has distributed tracing implemented.
You get out-of-the-box metrics from your tracing pipeline.

Even if you already have metrics, span metrics can provide in-depth monitoring of your system.
The generated metrics will show application level insight into your monitoring,
as far as tracing gets propagated through your applications.

## Example

<p align="center"><img src="../span-metrics-example.png" alt="Span metrics overview"></p>