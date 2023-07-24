---
title: 'Monitor'
description: Monitor TempoStack instances
menuTitle: Quickstart
weight: 300
aliases:
- /docs/tempo/operator/monitor
---

## Distributed Tracing

All Tempo components as well as the [Tempo Gateway](https://github.com/observatorium/api) support the export of traces in `thrift_compact` format.

### Configure tracing of Operands

#### Requirements

* An [OpenTelemetry Operator](https://opentelemetry.io/docs/k8s-operator/#getting-started) installation.
* *Optional:* Another tracing backend would be ideal - If none exists, a Jaeger instance can be created.

#### Installation

* Deploy the Tempo Operator to your cluster.
* Create an `OpenTelemetryCollector` CR that receives trace data in Jaeger Thrift format and exports data via OTLP to the desired trace backend.

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
