---
title: Troubleshoot Grafana Alloy
menuTitle: Grafana Alloy
description: Gain visibility on how many traces are being pushed to Grafana Alloy and if they are making it to the Tempo backend.
weight: 472
aliases:
- ../operations/troubleshooting/agent/
- ./agent.md # /docs/tempo/<TEMPO_VERSION>/troubleshooting/agent.md
---

# Troubleshoot Grafana Alloy

Sometimes it can be difficult to tell what, if anything, Grafana Alloy is sending along to the backend.
This document focuses on a few techniques to gain visibility on how many trace spans are pushed to Alloy and if they're making it to the backend.
[OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector) form the basis of the tracing pipeline, which
does a fantastic job of logging network and other issues.

If your logs are showing no obvious errors, one of the following suggestions may help.

## Metrics

Alloy publishes a few Prometheus metrics that are useful to determine how much trace traffic it receives and successfully forwards.
These metrics are a good place to start when diagnosing tracing Alloy issues.

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

### Check metrics in Grafana Cloud

If you are using Grafana Alloy to send traces to Grafana Cloud, the metrics are visible at
`http://localhost:12345/metrics`.
The `/metrics` HTTP endpoint of the Alloy HTTP server exposes the Alloy component and controller metrics.
Refer to the [Monitor the Grafana Alloy component controller](https://grafana.com/docs/alloy/latest/troubleshoot/controller_metrics/) documentation for more information.

In your Grafana Cloud instance, they can be checked in the `grafanacloud-usage` data source.
To view the metrics, use the following steps:

1. From your Grafana instance, select **Explore** in the left menu.
1. Change the data source to `grafanacloud-usage`.
1. Type the metric to verify in the text box. If you start with `grafanacloud_traces_`, you can  use autocomplete to browse the list of available metrics.

![Use Explore to check the metrics for traces sent to Grafana Cloud](/media/docs/tempo/screenshot-tempo-trouble-metrics-search.png)

## Trace span logging

If metrics and logs are looking good, but you are still unable to find traces in Grafana Cloud, you can configure Alloy to output all the traces it receives to the [console](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/grafana-alloy/automatic-logging/).
