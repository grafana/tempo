---
title: Monitor Tempo instances and the operator
description: Set up monitoring for Tempo instances and the operator
menuTitle: Monitor
weight: 300
aliases:
- /docs/tempo/operator/monitor
---

# Monitor Tempo instances and the operator

You can configure the Tempo Operator to monitor TempoStack instances (including all Tempo components like the distributor). In addition, the operator can expose metrics about the operator itself (for example, the number of successful and failed upgrades, etc.).


## Monitor TempoStack instances

The Tempo Operator supports monitoring and alerting of each Tempo component (distributor, ingester, etc.).
To enable metrics and alerting, the [Prometheus Operator](https://github.com/prometheus-operator/prometheus-operator) or a comparable solution which discovers `ServiceMonitor` and `PrometheusRule` objects must be installed and configured in the cluster.

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

### Configure distributed tracing of operands

All Tempo components as well as the [Tempo Gateway](https://github.com/observatorium/api) support the export of traces in `thrift_compact` format.

#### Deploy OpenTelemetry collector sidecar

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

#### Send trace data to OpenTelemetry sidecar

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


## Monitor the operator

The Tempo Operator can expose upgrade and other operational metrics about the operator itself, and can create alerts based on these metrics.
For example, the operator handles Tempo upgrades and exposes metrics like "the number of successful Tempo upgrades", "number of failed Tempo upgrades", and others.
The operator also creates alerts to notify system administrators if any Tempo upgrade fails.

Other metrics are internal to the operator itself, for example, the duration of a reconcile loop iteration.
This operator-specific component continuously tries to match the expected state as described in the TempoStack custom resource to the actual cluster state.
For example, if an object is deleted in the cluster which is managed by the operator, the operator re-creates this object again, to match the expected state of the cluster.

The operator can be configured using the ConfigMap `tempo-operator-manager-config` in the same namespace as the operator.
The following excerpt shows the configuration options to enable the creation of `ServiceMonitor` (for scraping metrics) and `PrometheusRule` (for creating alerts) objects:

```yaml
apiVersion: v1
kind: ConfigMap
data:
  controller_manager_config.yaml: |
    featureGates:
      observability:
        metrics:
          createServiceMonitors: true
          createPrometheusRules: true
```
