---
title: Choose where to generate metrics from traces
menuTitle: Generate locally or in Tempo
description: Generate span metrics before sampling so RED metrics, alerts, and service graphs still describe your full estate.
keywords:
  - metrics-generator
  - span metrics
  - sampling
  - Alloy
weight: 50
---

# Choose where to generate metrics from traces

Sampling reduces what Grafana Tempo stores.
It doesn't reduce what your services did.

Grafana Tempo's [metrics-generator](../) builds RED (rate, error, duration) metrics and service graphs from traces after ingest.
If a collector already dropped most of those traces, the metrics describe the sample, not the system.
Request rate looks low.
Error rate looks healthy.
Service graphs lose edges that never arrived.
Alerts that depend on those series miss the traffic you threw away.

You have four options.

## Use Tempo metrics-generator

Tempo metrics-generator creates span metrics and service graphs after traces are ingested.

Use it when traces reach Tempo unsampled.
One component does the work, collectors stay thin, and you don't emit the same series from every collector replica.
That's the cheaper path at scale.
Grafana Cloud Traces uses this model.

Refer to [Use the metrics-generator to create metrics from spans](../../span-metrics/span-metrics-metrics-generator/).
For Grafana Cloud Traces, refer to [Metrics-generator in Grafana Cloud](https://grafana.com/docs/grafana-cloud/send-data/traces/metrics-generator/).

## Generate metrics in Alloy or the OpenTelemetry Collector

Alloy and the OpenTelemetry Collector create the same class of metrics in the collector pipeline, before the sampler.

Use this when you tail sample before traces reach Tempo.
RED metrics and alerts still describe all traffic, not the leftover traces.
You cut storage without feeding dashboards a sample.

Alloy and the OpenTelemetry Collector are the same choice.
Use the collector you already run.

For Alloy settings, refer to [Use Alloy to generate metrics from spans](../../span-metrics/span-metrics-alloy/).
For OpenTelemetry Collector settings, refer to [Generate metrics from spans](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/instrument-send/set-up-collector/otel-collector/#generate-metrics-from-spans).
Put the connectors ahead of the sampler.
Refer to [Pipeline workflows](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/instrument-send/set-up-collector/tail-sampling/#pipeline-workflows).

## Avoid running both

You can run a collector generator and Tempo metrics-generator at the same time.
They write different series names, so they don't collide and they don't replace each other.

Don't do this unless you have a specific reason.
Two generators means two sets of active series, extra compute, and extra Grafana Cloud cost.

## Scale ratio-based samples in Tempo

A ratio-based sampler keeps a fixed fraction of traces and can record that fraction on the span.
Tempo multiplies the metric so counts match the unsampled population.

Use this when you sample at a fixed ratio, not when you tail sample.
You keep generation in Tempo and you don't need a collector-side generator.

Refer to [Handling sampled traces](../../span-metrics/span-metrics-metrics-generator/#handling-sampled-traces).
