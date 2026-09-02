---
title: Span metrics
description: Learn about span metrics in Grafana Tempo.
weight: 400
---

# Span metrics

Span metrics are generated from traces and can be used to create service graphs.
You can create span metrics in Tempo's metrics-generator, Grafana Alloy, or the OpenTelemetry Collector.

If you sample traces before they reach Tempo, choose where to generate those metrics first.
Refer to [Choose where to generate metrics from traces](../metrics-generator/generate-locally/).

The pages in this section cover Tempo and Alloy settings.
For OpenTelemetry Collector settings, including the pipeline diagram, refer to [Generate metrics from spans](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/instrument-send/set-up-collector/otel-collector/#generate-metrics-from-spans).

{{< section withDescriptions="true">}}
