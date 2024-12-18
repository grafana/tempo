---
aliases:
- /docs/tempo/latest/server_side_metrics/service_graphs/
- /docs/tempo/latest/metrics-generator/service_graphs/
title: Enable service graphs
description: Learn how to enable service graphs
weight: 200
refs:
  cardinality:
    - pattern: /docs/tempo/
      destination: https://grafana.com/docs/tempo/<TEMPO_VERSION>/metrics-generator/cardinality/
    - pattern: /docs/enterprise-traces/
      destination: https://grafana.com/docs/enterprise-traces/<ENTERPRISE_TRACES_VERSION>/metrics-generator/cardinality/
---

## Enable service graphs

Service graphs are generated in Tempo and pushed to a metrics storage.
Then, they can be represented in Grafana as a graph.
You need those components to fully use service graphs.

{{< admonition type="note" >}}
Cardinality can pose a problem when you have lots of services.
To learn more about cardinality and how to perform a dry run of the metrics-generator, refer to the [Cardinality documentation](ref:cardinality).
{{< /admonition >}}

### Enable service graphs in Tempo/GET

To enable service graphs in Tempo/GET, enable the metrics generator and add an overrides section which enables the `service-graphs` generator.
For more information, refer to the [configuration details](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration#metrics-generator).

To enable service graphs when using Grafana Alloy, refer to the [Grafana Alloy and service graphs documentation](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/grafana-alloy/service-graphs/).

### Enable service graphs in Grafana

{{< admonition type="note" >}}
Service graphs are enabled by default in Grafana. Prior to Grafana 9.0.4, service graphs were hidden
under the [feature toggle](/docs/grafana/latest/setup-grafana/configure-grafana) `tempoServiceGraph`.
{{< /admonition >}}

Configure a Tempo data source's service graphs by linking to the Prometheus backend where metrics are being sent:

```
apiVersion: 1
datasources:
  # Prometheus backend where metrics are sent
  - name: Prometheus
    type: prometheus
    uid: prometheus
    url: <prometheus-url>
    jsonData:
        httpMethod: GET
    version: 1
  - name: Tempo
    type: tempo
    uid: tempo
    url: <tempo-url>
    jsonData:
      httpMethod: GET
      serviceMap:
        datasourceUid: 'prometheus'
    version: 1
```