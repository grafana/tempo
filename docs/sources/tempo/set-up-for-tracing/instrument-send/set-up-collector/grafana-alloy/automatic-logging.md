---
title: "Automatic logging: trace discovery through logs"
description: Automatic logging provides an easy and fast way of getting trace discovery through logs.
menuTitle: Automatic logging
weight: 200
aliases:
  - ../../../../grafana-alloy/automatic-logging/ # /docs/tempo/latest/grafana-alloy/automatic-logging/
  - ../../../../configuration/grafana-alloy/automatic-logging/ # /docs/tempo/latest/configuration/grafana-alloy/automatic-logging/
  - ../../../../configuration/grafana-agent/automatic-logging/ # /docs/tempo/latest/configuration/grafana-agent/automatic-logging/
---

# Automatic logging: trace discovery through logs

When you run instrumented distributed systems, discovering which traces exist can be challenging.
Grafana Alloy's automatic logging solves this by writing well-formatted log lines for each span, root, or process that passes through the tracing pipeline.

Automatic logging lets you discover trace IDs through log messages.
You can search for traces by key-value pairs in Loki and jump from a log message directly to the trace view in Grafana.

While this approach is useful, it isn't as powerful as TraceQL.
If you want to log the trace ID to enable moving from logs to traces, read on.
To query traces directly, refer to the [TraceQL documentation](https://grafana.com/docs/tempo/<TEMPO_VERSION>/traceql/).

## Before you begin

To use automatic logging, you need:

- [Grafana Alloy](https://grafana.com/docs/alloy/<ALLOY_VERSION>/) installed and receiving traces from your application.
- A Loki instance to store the generated logs.
- A Tempo instance to store traces so you can navigate from logs to traces.
- Grafana configured with both a [Loki data source](https://grafana.com/docs/grafana/<GRAFANA_VERSION>/datasources/loki/) and a [Tempo data source](https://grafana.com/docs/grafana/<GRAFANA_VERSION>/datasources/tempo/).

## Configure automatic logging

Automatic logging uses the `otelcol.connector.spanlogs` connector to generate log lines from trace spans.
The connector accepts traces and generates logs, but it doesn't forward the original traces.
You must send traces to both the `spanlogs` connector and your trace backend to avoid losing trace data.

![Automatic logging pipeline showing traces fanning out from the OTLP receiver to both the spanlogs connector and the trace exporter.](../automatic-logging-pipeline.svg)

For high-throughput systems, logging every span may generate too much volume.
In those cases, log per root span or process instead.

The connector searches for a set of span or resource attributes in the spans and logs them as key-value pairs.
You can then search by those key-value pairs in Loki.

For all available configuration options, refer to the `otelcol.connector.spanlogs` [component reference](https://grafana.com/docs/alloy/<ALLOY_VERSION>/reference/components/otelcol/otelcol.connector.spanlogs/).

### Log roots to an OTLP endpoint

This example logs root spans and sends the generated logs to an OTLP endpoint.
Traces are forwarded to the same endpoint so they aren't lost:

```alloy
otelcol.receiver.otlp "default" {
  grpc {}
  http {}

  output {
    traces = [
      otelcol.connector.spanlogs.default.input,
      otelcol.exporter.otlp.default.input,
    ]
  }
}

otelcol.connector.spanlogs "default" {
  roots = true

  output {
    logs = [otelcol.exporter.otlp.default.input]
  }
}

otelcol.exporter.otlp "default" {
  client {
    endpoint = env("OTLP_ENDPOINT")
  }
}
```

### Log roots with custom attributes to Loki

This example logs root spans with `http.method` and `http.target` attributes, then sends the generated logs to a Loki instance.
Traces are forwarded separately to a Tempo instance.

Because `otelcol.exporter.loki` doesn't promote log attributes to Loki labels by default,
you must use `otelcol.processor.attributes` to add a `loki.attribute.labels` hint.
This promotes the `traces` attribute to a Loki label so you can filter by log type:

```alloy
otelcol.receiver.otlp "default" {
  grpc {}
  http {}

  output {
    traces = [
      otelcol.connector.spanlogs.default.input,
      otelcol.exporter.otlp.tempo.input,
    ]
  }
}

otelcol.connector.spanlogs "default" {
  roots           = true
  span_attributes = ["http.method", "http.target"]

  output {
    logs = [otelcol.processor.attributes.default.input]
  }
}

otelcol.processor.attributes "default" {
  action {
    key    = "loki.attribute.labels"
    action = "insert"
    value  = "traces"
  }

  output {
    logs = [otelcol.exporter.loki.default.input]
  }
}

otelcol.exporter.loki "default" {
  forward_to = [loki.write.local.receiver]
}

loki.write "local" {
  endpoint {
    url = "http://loki:3100/loki/api/v1/push"
  }
}

otelcol.exporter.otlp "tempo" {
  client {
    endpoint = "tempo:4317"
  }
}
```

## Expected output

When you enable automatic logging, the connector generates a log line for each span, root, or process depending on which options you enable.
Each log line uses a `logfmt`-style body with the following default keys:

| Key      | Description                                                                                                                                            |
| -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `svc`    | The service name from the span's resource.                                                                                                             |
| `span`   | The name of the span.                                                                                                                                  |
| `dur`    | The duration of the span in nanoseconds, for example, `150200000ns`.                                                                                   |
| `tid`    | The trace ID.                                                                                                                                          |
| `status` | The status of the span. Only included when the status is explicitly set (not `STATUS_CODE_UNSET`). Values are `STATUS_CODE_OK` or `STATUS_CODE_ERROR`. |

You can add more keys by configuring `span_attributes`, `process_attributes`, or `event_attributes`.
You can customize all key names using the `overrides` block.
For details, refer to the `otelcol.connector.spanlogs` [component reference](https://grafana.com/docs/alloy/<ALLOY_VERSION>/reference/components/otelcol/otelcol.connector.spanlogs/).

For example, a root span log line might look like this:

```
span="HTTP GET" dur=150200000ns http.method=GET http.target=/api/v1/query svc=my-service tid=7bba9f33312b3dbb8b2c2c62bb7abe2d
```

Each log line also has a `traces` attribute that indicates the log type: `span`, `root`, `process`, or `event`.
If you configured the `loki.attribute.labels` hint as shown in the Loki example, this attribute becomes a Loki label that you can filter on.

### Query automatic logging data in Loki

Use LogQL in Grafana Explore to query the generated logs.

To find all root span logs:

```logql
{traces="root"}
```

To filter for slow requests from a specific service:

```logql
{traces="root"} | logfmt | dur > 2s and svc="my-service"
```

### Navigate from logs to traces

To link from a log line directly to its trace in Tempo, configure [derived fields](https://grafana.com/docs/grafana/<GRAFANA_VERSION>/datasources/loki/configure-loki-data-source/#derived-fields) on your Loki data source in Grafana:

1. Go to **Connections** > **Data sources** and select your Loki data source.
1. In **Derived fields**, add a new field with these settings:
   - **Name**: `TraceID`
   - **Type**: **Label**
   - **Match field name**: `tid`
   - **Internal link** enabled, pointing to your Tempo data source
1. Save the data source configuration.

After this configuration, log results in Explore display a link next to the `tid` field.
