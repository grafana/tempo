---
title: Monitor TempoStack instances
description: Monitor TempoStack instances
menuTitle: Monitor
weight: 300
aliases:
- /docs/tempo/operator/monitor
---

# Monitor TempoStack instances

The configuration for monitoring `TempoStack` instances is exposed in the CR:

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

## Configure distributed tracing of operands

All Tempo components as well as the [Tempo Gateway](https://github.com/observatorium/api) support the export of traces in `thrift_compact` format.

### Deploy OpenTelemetry collector sidecar

To deploy the OpenTelemetry collector, follow these steps:
1. Install [OpenTelemetry Operator](https://opentelemetry.io/docs/k8s-operator/#getting-started) into the cluster.
2. Create an `OpenTelemetryCollector` CR that receives trace data in Jaeger Thrift format and exports data via OTLP to the desired trace backend.
3. **Optional:** Deploy tracing backend to store trace data.

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

### Send trace data to OpenTelemetry sidecar

Finally, create a `TempoStack` instance that sets `jaeger_agent_endpoint` to report trace data to the `localhost`. 
The Tempo operator sets the OpenTelemetry inject annotation `sidecar.opentelemetry.io/inject": "true` to all `TempoStack` pods.
The OpenTelemetry Operator will recognize the annotation, and it will inject a sidecar into all `TempoStack` pods.

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
