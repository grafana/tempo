---
title: Automatic logging
weight: 200
aliases:
- /docs/tempo/grafana-agent/automatic-logging
---

# Automatic logging: Trace discovery through logs

Running distributed tracing systems is very powerful, but it brings its own challenges,
and one of them is trace discovery.
Tempo supports finding a trace if you know the trace identifier,
so we leverage other tools like logs and metrics to discover traces.

Automatic logging provides an easy and fast way of getting trace discovery through logs.
Automatic logging writes a well formatted log line to a Loki instance or to stdout for each span, root or process that passes through the tracing pipeline.
This allows for automatically building a mechanism for trace discovery.
On top of that, we also get metrics from traces using Loki.

For high throughput systems, logging for every span may generate too much volume.
In such cases, logging per root span or process is recommended.

<p align="center"><img src="../automatic-logging.png" alt="Automatic logging overview"></p>

Automatic logging searches for a given set of attributes in the spans and logs them as key-value pairs.
This allows searching by those key-value pairs in Loki.

## Before you begin

To configure automatic logging, you need to select your preferred backend and the trace data to log.

To see all the available config options, refer to the [configuration reference]({{< relref "/docs/agent/latest/configuration/traces-config" >}}).

This simple example logs trace roots to stdout and is a good way to get started using automatic logging:
```yaml
traces:
  configs:
  - name: default
    ...
    automatic_logging:
      backend: stdout
      roots: true
```

This example pushes logs directly to a Loki instance also configured in the same Grafana Agent.

```yaml
traces:
  configs:
  - name: default
    ...
    automatic_logging:
      backend: logs_instance
      logs_instance_name: default
      roots: true
```

## Examples

<p align="center"><img src="../automatic-logging-example-query.png" alt="Automatic logging overview"></p>
<p align="center"><img src="../automatic-logging-example-results.png" alt="Automatic logging overview"></p>
