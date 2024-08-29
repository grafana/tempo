---
title: Troubleshoot Grafana Alloy
menuTitle: Grafana Alloy
description: Gain visibility on how many traces are being pushed to Grafana Alloy and if they are making it to the Tempo backend.
weight: 472
aliases:
- ../operations/troubleshooting/agent/
---

# Troubleshoot Grafana Alloy

Sometimes it can be difficult to tell what, if anything, Grafana Alloy is sending along to the backend.
This document focuses on a few techniques to gain visibility on how many trace spans are pushed to Alloy and if they're making it to the backend.
[OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector) form the basis of the tracing pipeline, which
does a fantastic job of logging network and other issues.

If your logs are showing no obvious errors, one of the following suggestions may help.

## Metrics

Alloy publishes a few Prometheus metrics that are useful to determine how much trace traffic it receives and successfully forwards.
These metrics are a good place to start when diagnosing tracing Grafana Alloy issues.

From the [`otelcol.receiver.otlp`](https://grafana.com/docs/alloy/<ALLOY_LATEST>/reference/components/otelcol/otelcol.receiver.otlp/) component:
```
receiver_accepted_spans_ratio_total
receiver_refused_spans_ratio_total
```

From the [`otelcol.exporter.otlp`](https://grafana.com/docs/alloy/<ALLOY_LATEST>/reference/components/otelcol/otelcol.exporter.otlp/) component:
```
exporter_sent_spans_ratio_total
exporter_send_failed_spans_ratio_total
```

## Trace span logging

If metrics and logs are looking good, but you are still unable to find traces in Grafana Cloud, you can configure Alloy to output all the traces it receives to the [console](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/grafana-alloy/automatic-logging/).
