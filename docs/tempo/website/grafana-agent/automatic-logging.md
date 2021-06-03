---
title: Automatic logging
---

# Automatic logging: Trace discovery through logs

Running distributed tracing systems is very powerful, but it brings its own challenges,
and one of them is trace discovery.
Tempo supports finding a trace if you know the trace identifier,
so we leverage other tools like logs and metrics to discover traces.

Automatic logging provides an easy and fast way of getting trace discovery through logs.
Automatic logging writes a well formatted log line to a Loki instance for each span, root or process that passes through the tracing pipeline.
This allows for automatically building a mechanism for trace discovery.
On top of that, we also get metrics from traces using Loki.

<p align="center"><img src="../automatic-logging.png" alt="Automatic logging overview"></p>

Automatic logging searches for a given set of attributes in the spans and logs them as key-value pairs.
This allows to search by those key-value pairs in Loki.

Automatic logging also supports logging to stdout.

## Quick start

To configure it, you need to select your preferred backend and what trace data to log.
To see all the available config options, refer to the [configuration reference](https://github.com/grafana/agent/blob/main/docs/configuration-reference.md#tempo_instance_config).

```
tempo:
  configs:
  - name: default
    ...
    automatic_logging:
      backend: loki
      loki_name: default
      spans: true
```
