---
title: Service graphs
---

# Service graphs

<span style="background-color:#f3f973;">This is an experimental feature. For more information about how to enable it, continue reading.</span>

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

<p align="center"><img src="../service-graphs.png" alt="Service graphs example"></p>

## How they work

The metrics-generator will process traces and generate service graphs in the form of prometheus metrics.

Service graphs work by inspecting traces and looking for spans with parent-children relationship that represent a request.
Additionally, spans need to contain the tag [`span.kind`](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/trace/api.md#spankind),
with value `CLIENT` for the parent span (i.e. service that started the request) and `SERVER` for the children span (i.e. service that received the request).

If these conditions are met, those spans are recorded as an edge in the graph, represented by a metric.
The nodes in the graphs, or services, are indicated in that metric by the labels `client` and `server`.

```
  tempo_service_graph_request_total{client="app", server="db"} 20
```

Every span that can be paired to form a request is kept in an in-memory store,
until its corresponding pair span is received or the maximum waiting time has passed.
When either of these conditions is reached, the request is recorded and removed from the local store.

### Metrics

The following metrics are exported:

| Metric                                      | Type      | Labels         | Description                                                  |
|---------------------------------------------|-----------|----------------|--------------------------------------------------------------|
| traces_service_graph_request_total          | Counter   | client, server | Total count of requests between two nodes                    |
| traces_service_graph_request_failed_total   | Counter   | client, server | Total count of failed requests between two nodes             |
| traces_service_graph_request_server_seconds | Histogram | client, server | Time for a request between two nodes as seen from the server |
| traces_service_graph_request_client_seconds | Histogram | client, server | Time for a request between two nodes as seen from the client |
| traces_service_graph_unpaired_spans_total   | Counter   | client, server | Total count of unpaired spans                                |
| traces_service_graph_dropped_spans_total    | Counter   | client, server | Total count of dropped spans                                 |

Duration is measured both from the client and the server sides.

Since the service graph processor has to process both sides of an edge,
it needs to process all spans of a trace to function properly.
If spans of a trace are spread out over multiple instances it will not be possible to pair up spans reliably.

## Cardinality

Cardinality can pose a problem when you have lots of services.
There isn't a direct formula or solution to this issue.
But the following guide should help estimate the cardinality that the feature will generate.

### How to estimate the cardinality

#### Cardinality from traces

The amount of edges depends on the amount of nodes in the system and the direction of the requests between them.
Let’s call this amount hops. Every hop will be a unique combination of client + server labels.

For example:
- a system with 3 nodes `(A, B, C)` of which A only calls B and B only calls C will have 2 hops `(A → B, B → C)`
- a system with 3 nodes `(A, B, C)` that call each other (i.e. all bidirectional links somehow) will have 6 hops `(A → B, B → A, B → C, C → B, A → C, C → A)`

We can’t calculate the amount of hops automatically based upon the nodes,
but it should be a value between `#services - 1` and `#services!`.

If we know the amount of hops in a system, we can calculate the cardinality of the generated service graphs:

```
  traces_service_graph_request_total: #hops
  traces_service_graph_request_failed_total: #hops
  traces_service_graph_request_server_seconds: 3 buckets * #hops
  traces_service_graph_request_client_seconds: 3 buckets * #hops
  traces_service_graph_unpaired_spans_total: #services (absolute worst case)
  traces_service_graph_dropped_spans_total: #services (absolute worst case)
```

Finally, we get the following cardinality estimation:

```
  Sum: 8 * #hops + 2 * #services
```

#### Dry-running the metrics-generator

An often most reliable solution is by running the metrics-generator in a dry-run mode.
That is generating metrics but not collecting them, thus not writing them to a metrics storage.
The override `metrics_generator_disable_collection` is defined for this use-case.

To get an estimate, run the metrics-generator normally and set the override to `false`.
Then, check `tempo_metrics_generator_registry_active_series` to get an estimation of the active series for that set-up.

## How to run

Service graphs are generated in Tempo and pushed to a metrics storage.
Then, they can be represented in Grafana as a graph.
You will need those components to fully use service graphs.

### Tempo

<!-- TODO: Link to operations folder -->

### Grafana

Service graphs is hidden under the feature flag `tempoServiceGraph`.

To run this feature:
1. Run Grafana 8.2 or the latest pre-release and enable the `tempoServiceGraph` [feature toggle](https://grafana.com/docs/grafana/latest/packages_api/data/featuretoggles/#temposervicegraph-property).
2. Configure a Tempo datasource's 'Service Graphs' section by linking to the prometheus backend where metrics are being sent.

Example provisioned datasource config for service graphs:

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

