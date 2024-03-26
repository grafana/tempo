---
title: Monolithic deployment
description: Set up a Tempo deployment in monolithic mode
menuTitle: Monolithic deployment
weight: 500
aliases:
- /docs/tempo/operator/monolithic
---

# Monolithic deployment

The `TempoMonolithic` Custom Resource (CR) creates a Tempo deployment in [Monolithic mode](https:grafana.com/docs/tempo/<TEMPO_VERSION>/setup/deployment/#monolithic-mode).
In this mode, all components of the Tempo deployment (compactor, distributor, ingester, querier and query-frontend) are contained in a single container.

This type of deployment is ideal for small deployments, demo and test setups, and supports storing traces in memory, in a Persistent Volume and in object storage.

{{< admonition type="note" >}}
The monolithic deployment of Tempo does not scale horizontally. If you require horizontal scaling, please use the `TempoStack` CR for a Tempo deployment in [Microservices mode](https://grafana.com/docs/tempo/<TEMPO_VERSION>/setup/deployment/#microservices-mode).
{{< /admonition >}}

## Quickstart

The following manifest creates a Tempo monolithic deployment with trace ingestion over OTLP/gRPC and OTLP/HTTP, storing traces in a 2 GiB tmpfs volume (in-memory storage).

```yaml
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoMonolithic
metadata:
  name: sample
spec:
  storage:
    traces:
      backend: memory
      size: 2Gi
```

Once the pod is ready, you can send traces to `tempo-sample:4317` (OTLP/gRPC) and `tempo-sample:4318` (OTLP/HTTP) inside the cluster.

To configure a Grafana data source, use the URL `http://tempo-sample:3200` (available inside the cluster).

## CRD Specification
A manifest with all available configuration options is available here: [tempo.grafana.com_tempomonolithics.yaml](https://github.com/grafana/tempo-operator/blob/main/docs/spec/tempo.grafana.com_tempomonolithics.yaml).

{{< admonition type="note" >}}
This file is auto-generated and does not constitute a valid CR.
{{< /admonition >}}

It provides an overview of the structure, the available configuration options and help texts.
