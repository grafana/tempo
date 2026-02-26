---
aliases:
  - ../server_side_metrics/service_graphs/ # /docs/tempo/next/server_side_metrics/service_graphs/
  - ../metrics-generator/service_graphs/ # /docs/tempo/next/metrics-generator/service_graphs/
title: Service graphs
description: Service graphs help you understand the structure of a distributed system and the connections and dependencies between its components.
weight: 500
---

# Service graphs

A service graph is a visual representation of the interrelationships between various services.
Service graphs help you to understand the structure of a distributed system,
and the connections and dependencies between its components:

- **Infer the topology of a distributed system.**
  As distributed systems grow, they become more complex.
  Service graphs help you to understand the structure of the system.
- **Provide a high-level overview of the health of your system.**
  Service graphs display error rates, latencies, as well as other relevant data.
- **Provide an historic view of a system’s topology.**
  Distributed systems change very frequently,
  and service graphs offer a way of seeing how these systems have evolved over time.

Service graphs can be generated from metrics created by the metrics-generator or Grafana Alloy.
Refer to [Enable service graphs](/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/service_graphs/enable-service-graphs/) for more information on how to enable service graphs in Tempo.

![Service graph](/media/docs/grafana/data-sources/tempo/query-editor/tempo-ds-query-service-graph-prom.png)

## How they work

The metrics-generator and Grafana Alloy both process traces and generate service graphs in the form of Prometheus metrics.

Service graphs work by inspecting traces and looking for spans with parent-children relationship that represent a request.
The processor uses the [OpenTelemetry semantic conventions](https://github.com/open-telemetry/semantic-conventions/blob/main/docs/general/trace.md) to detect a myriad of requests.

It supports the following requests:

- A direct request between two services where the outgoing and the incoming span must have [`span.kind`](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/trace/api.md#spankind), `client`, and `server`, respectively.
- A request across a messaging system where the outgoing and the incoming span must have `span.kind`, `producer`, and `consumer` respectively.
- A database request; in this case the processor looks for spans containing attributes `span.kind`=`client` as well as one of `db.namespace`, `db.name` or `db.system`. See below for how the name of the node is determined for a database request.

The processor keeps every span that can form a request pair in an in-memory store until the corresponding pair span arrives or the maximum waiting time passes.
When either condition occurs, the processor records the request and removes it from the local store.

Each emitted metrics series have the `client` and `server` label corresponding with the service doing the request and the service receiving the request.

```
  traces_service_graph_request_total{client="app", server="db", connection_type="database"} 20
```

### Virtual nodes

Virtual nodes are nodes that form part of the lifecycle of a trace,
but spans for them aren't collected because they're outside the user's reach or aren't instrumented.
For example, you might not collect spans for an external service for payment processing that's outside user interaction.

The processor detects virtual nodes in two ways:

- **Uninstrumented client (missing client span):** The root span has `span.kind` set to `server` or `consumer`, with no matching client span. This indicates that the request or message was initiated by an external system that isn't instrumented, like a scheduler, a frontend application, or an engineer using `curl`.
  - In the Tempo metrics-generator, the processor checks the configured `peer_attributes` on the server span first. If it finds a matching attribute, it uses that value as the client node name. Otherwise, the client node name defaults to `user`.
  - In Grafana Alloy and the OpenTelemetry Collector `servicegraph` connector, the connector doesn't evaluate peer attributes for this case. The client node name always defaults to `user` and you can't override it. An [upstream feature request](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/45397) exists to add this capability.
- **Uninstrumented server (missing server span):** A `client` span doesn't have its matching `server` span, but has a peer attribute present. In this case, the client called an external service that doesn't send spans. The processor uses the peer attribute value as the virtual server node name.
  - The default peer attributes are `peer.service`, `db.name`, and `db.system`.
  - The processor searches the attributes in order and uses the first match as the virtual node name.

The processor identifies a database node when the span has at least one `db.namespace`, `db.name`, or `db.system` attribute.

The processor determines the database node name using the following span attributes in order of precedence: `peer.service`, `server.address`, `network.peer.address:network.peer.port`, `db.namespace`, `db.name`.

### Metrics

The following metrics are exported:

<!-- vale Grafana.Spelling = NO -->

| Metric                                                  | Type      | Labels                          | Description                                                                                                |
| ------------------------------------------------------- | --------- | ------------------------------- | ---------------------------------------------------------------------------------------------------------- |
| `traces_service_graph_request_total`                    | Counter   | client, server, connection_type | Total count of requests between two nodes                                                                  |
| `traces_service_graph_request_failed_total`             | Counter   | client, server, connection_type | Total count of failed requests between two nodes                                                           |
| `traces_service_graph_request_server_seconds`           | Histogram | client, server, connection_type | Time for a request between two nodes as seen from the server                                               |
| `traces_service_graph_request_client_seconds`           | Histogram | client, server, connection_type | Time for a request between two nodes as seen from the client                                               |
| `traces_service_graph_request_messaging_system_seconds` | Histogram | client, server, connection_type | (Off by default) Time between publisher and consumer for services communicating through a messaging system |
| `traces_service_graph_unpaired_spans_total`             | Counter   | client, server, connection_type | Total count of unpaired spans                                                                              |
| `traces_service_graph_dropped_spans_total`              | Counter   | client, server, connection_type | Total count of dropped spans                                                                               |

<!-- vale Grafana.Spelling = YES -->

The processor measures duration from both the client and server sides.

Possible values for `connection_type`: unset, `virtual_node`, `messaging_system`, or `database`.

You can include additional labels using the `dimensions` configuration option or the `enable_virtual_node_label` option.

Since the service graph processor has to process both sides of an edge,
it needs to process all spans of a trace to function properly.
If spans of a trace spread across multiple instances, the processor can't pair them reliably.

#### Activate `enable_virtual_node_label`

Activating this feature adds the following label and corresponding values:

| Label          | Possible Values             | Description                                  |
| -------------- | --------------------------- | -------------------------------------------- |
| `virtual_node` | `unset`, `client`, `server` | Explicitly indicates the uninstrumented side |
