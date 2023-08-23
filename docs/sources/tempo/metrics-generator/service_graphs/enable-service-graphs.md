---
aliases:
- /docs/tempo/latest/server_side_metrics/service_graphs/
- /docs/tempo/latest/metrics-generator/service_graphs/
title: Enable service graphs
description: Learn how to enable service graphs
weight: 300
---


## Enable service graphs

Service graphs are generated in Tempo and pushed to a metrics storage.
Then, they can be represented in Grafana as a graph.
You will need those components to fully use service graphs.

{{% admonition type="note" %}}
Cardinality can pose a problem when you have lots of services.
To learn more about cardinality and how to perform a dry run of the metrics generator, see the [Cardinality documentation]({{< relref "../cardinality" >}}).
{{% /admonition %}}

### Enable service graphs in Tempo/GET

To enable service graphs in Tempo/GET, enable the metrics generator and add an overrides section which enables the `service-graphs` generator.
For more information, refer to the [configuration details]({{< relref "../../configuration#metrics-generator" >}}).

### Enable service graphs in Grafana

{{% admonition type="note" %}}
Since Grafana 9.0.4, service graphs have been enabled by default. Prior to Grafana 9.0.4, service graphs were hidden
under the [feature toggle](/docs/grafana/latest/setup-grafana/configure-grafana/#feature_toggles) `tempoServiceGraph`.
{{% /admonition %}}

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