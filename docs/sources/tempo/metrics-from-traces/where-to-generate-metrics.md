---
title: Choose where to generate metrics from traces
menuTitle: Where to generate metrics
description: Learn how to choose between generating span metrics and service graphs in Tempo or in the collector pipeline.
keywords:
  - metrics-generator
  - span metrics
  - sampling
  - Alloy
weight: 50
---

# Choose where to generate metrics from traces

You can generate metrics from traces in Grafana Tempo or in the collector pipeline.
These metrics include RED metrics (rate, error, and duration) for request volume, failures, and latency,
and service graphs that map how services call each other.
Generating metrics from traces lets you build dashboards and alerts from your tracing pipeline
without a separate metrics instrumentation path.

Where you generate the metrics depends on how you sample traces before they reach Tempo.
Sampling reduces the traces Tempo stores.
It doesn't change the traffic your services produced.
Refer to [Sampling](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/instrument-send/set-up-collector/tail-sampling/) for head and tail sampling.

The [metrics-generator](/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/metrics-generator/) builds span metrics and service graphs from traces after ingest.
Span metrics are request counts, durations, and error rates derived from individual spans.
Service graphs are maps of relationships between services, built from parent-child span pairs.

If a collector already dropped most of those traces, the metrics describe the sample, not the full system.
Request rate, error rate, and latency no longer match the traffic your services produced.
Service graphs miss relationships that never arrived.
Alerts that depend on those series miss discarded traffic.

If you send traces to Grafana Cloud Traces, you can use [Adaptive Traces](https://grafana.com/docs/grafana-cloud/cost-management-and-billing/adaptive-telemetry/adaptive-traces/) instead of building and operating your own tail sampling pipeline.
Adaptive Traces is a managed tail sampling capability that analyzes your trace data, recommends sampling policies (for example, keep traces with errors or high latency), and applies them for you. Adaptive Traces can also generate metrics from all received traces.

## Choose a generation path

Use the table to choose a generation path, then follow the matching section.

| Option | Use when | Tradeoff |
| --- | --- | --- |
| [Tempo metrics-generator](#use-the-metrics-generator) | Traces reach Tempo unsampled | One component generates metrics and collectors stay thin. Metrics include only the traces Tempo ingested. |
| [Alloy or OpenTelemetry Collector](#generate-metrics-in-alloy-or-the-opentelemetry-collector) | You tail sample before traces reach Tempo | RED metrics and service graphs describe all traffic. Each collector replica emits series. |
| [Both generators](#avoid-running-both) | You have a specific requirement for two generation paths | Duplicate active series, extra compute, and extra cost. |
| [Scale ratio-based samples in Tempo](#scale-ratio-based-samples-in-tempo) | You sample at a fixed ratio and record that ratio on the span | Generation stays in Tempo. This option doesn't apply to tail sampling. |

## Use the metrics-generator

The metrics-generator creates span metrics and service graphs after traces are ingested.

Use it when traces reach Tempo unsampled.
One component generates the metrics, so collectors stay thin and you don't emit the same series from every collector replica.
That reduces active series and compute compared with generating metrics on every collector.
Grafana Cloud Traces uses this model.

Refer to [Use the metrics-generator to create metrics from spans](/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/span-metrics/span-metrics-metrics-generator/).
For Grafana Cloud Traces, refer to [Metrics-generator in Grafana Cloud](https://grafana.com/docs/grafana-cloud/send-data/traces/metrics-generator/).

## Generate metrics in Alloy or the OpenTelemetry Collector

Grafana Alloy and the OpenTelemetry Collector can create span metrics and service graphs in the collector pipeline, before sampling.

Use this option when you tail sample before traces reach Tempo.
Tail sampling evaluates a complete trace before deciding whether to keep or drop it.
Generating metrics before the sampler keeps RED metrics and alerts aligned with all traffic, not only the traces Tempo stores.

Alloy and the OpenTelemetry Collector provide the same generation capability in this context.
Use the collector you already run.

To generate metrics in the collector:

1. Configure the span metrics and service graph connectors in Alloy or the OpenTelemetry Collector.
1. Place the connectors ahead of the sampler so they see every span.

For Alloy settings, refer to [Use Alloy to generate metrics from spans](/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/span-metrics/span-metrics-alloy/).
For OpenTelemetry Collector settings, refer to [Generate metrics from spans](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/instrument-send/set-up-collector/otel-collector/#generate-metrics-from-spans).
For pipeline order, refer to [Pipeline workflows](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/instrument-send/set-up-collector/tail-sampling/#pipeline-workflows).

## Avoid running both

You can run a collector generator and Tempo metrics-generator at the same time.
The two paths don't replace each other.
Each path writes its own series, so you can end up with two sets of active series.

Running both generators increases active series, compute, and Grafana Cloud cost.
Use a single generation path unless you have a specific requirement.

If you already run both, look for two sets of span metrics or service graph series in your metrics backend.
Disable one path: turn off the metrics-generator processors in Tempo,
or remove the span metrics and service graph connectors from the collector pipeline.

## Scale ratio-based samples in Tempo

Ratio-based sampling keeps a fixed fraction of traces and can record that fraction on the span.
When configured, Tempo multiplies the metric so counts match the unsampled population.

Use this option when you sample at a fixed ratio, not when you tail sample.
Generation stays in Tempo, and you don't need a collector-side generator.

Refer to [Handling sampled traces](/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/span-metrics/span-metrics-metrics-generator/#handling-sampled-traces).
