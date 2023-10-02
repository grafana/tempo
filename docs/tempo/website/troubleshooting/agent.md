---
aliases:
- /docs/tempo/v1.2.1/troubleshooting/agent/
title: Agent Troubleshooting
weight: 472
---

# Agent Troubleshooting

Sometimes it can be difficult to tell what, if anything, the Grafana Agent is sending along to the backend. This document focuses
on a few techniques to gain visibility on how many traces are being pushed to the Agent and if they're making it to the
backend. The tracing pipeline is built on top of the [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector) which
does a fantastic job of logging network and other issues.

If your logs are showing no obvious errors try the following:

## Metrics

The agent publishes a few Prometheus metrics that are useful to determine how much trace traffic it is receiving and successfully forwarding. These
are a good place to start when diagnosing tracing Agent issues.

```
traces_receiver_accepted_spans
traces_receiver_refused_spans
traces_exporter_sent_spans
traces_exporter_send_failed_spans
```

## Automatic Logging

If metrics and logs are looking good, but you are still unable to find traces in Grafana Cloud then we can turn on [Automatic Logging]({{< relref "../grafana-agent/automatic-logging" >}}). A recommend debug setup is:

```
traces:
  configs:
  - name: default
    ...
    automatic_logging:
      backend: stdout
      roots: true
```

This will emit logs to stdout for every root span that the Agent forwards. This can be useful to see exactly which traces are being forwarded to Grafana
Cloud.