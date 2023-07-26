---
title: Monitor
description: Monitor TempoStack instances
menuTitle: Monitor
weight: 300
aliases:
- /docs/tempo/operator/monitor
---

# Monitor

Tempo operator and `TempoStack` operands can be monitored.
The monitoring configuration for operands is exposed the `TempoStack` CR:

```yaml
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoStack
spec:
  observability:
    metrics:
      createServiceMonitors: true
      createPrometheusRules: true
    tracing:
      sampling_fraction: 1.0
      jaeger_agent_endpoint: localhost:6831
```

## Distributed Tracing

All Tempo components as well as the [Tempo Gateway](https://github.com/observatorium/api) support the export of traces in `thrift_compact` format.

### Configure distributed tracing of operands

#### Deploy OpenTelemetry collector

* Deploy [OpenTelemetry Operator](https://opentelemetry.io/docs/k8s-operator/#getting-started) installation.
* Create an `OpenTelemetryCollector` CR that receives trace data in Jaeger Thrift format and exports data via OTLP to the desired trace backend.
* *Optional:* Deploy tracing backend to store trace data.

```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: OpenTelemetryCollector
metadata:
  name: sidecar-for-tempo
spec:
  mode: sidecar
  config: |
    receivers:
      jaeger:
        protocols:
          thrift_compact:

    exporters:
      otlp:
        endpoint: <otlp-endpoint>:4317
        tls:
          insecure: true

    service:
      pipelines:
        traces:
          receivers: [jaeger]
          exporters: [otlp]
```

#### Configuration

Finally, create a `TempoStack` instance that configures `jaeger_agent_endpoint` to report trace data to the `localhost`. 
The Tempo operator sets the inject annotation `sidecar.opentelemetry.io/inject": "true` to all `TempoStack` pods.
The annotation instructs the OpenTelemetry Operator to inject a sidecar into all `TempoStack` pods.

```yaml
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoStack
metadata:
  name: simple-stack
spec:
  template:
    queryFrontend:
      jaegerQuery:
        enabled:
  storage:
    secret:
      type: s3
      name: minio-test
  storageSize: 200M
  observability:
    tracing:
      sampling_fraction: "1.0"
      jaeger_agent_endpoint: localhost:6831
```
