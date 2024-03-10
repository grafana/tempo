---
title: Estimate cardinality from traces
menuTitle: Estimate cardinality
description: Service graphs help you understand the structure of a distributed system and the connections and dependencies between its components.
weight: 300
---

## Estimate cardinality from traces

Cardinality can pose a problem when you have lots of services.
There isn't a direct formula or solution to this issue.
The following guide should help estimate the cardinality that the feature will generate.

For more information on cardinality, refer to the [Cardinality]({{< relref "../cardinality" >}}) documentation.

### How to estimate the cardinality

The amount of edges depends on the number of nodes in the system and the direction of the requests between them.
Let’s call this amount hops. Every hop will be a unique combination of client + server labels.

For example:

- A system with 3 nodes `(A, B, C)` of which A only calls B and B only calls C will have 2 hops `(A → B, B → C)`
- A system with 3 nodes `(A, B, C)` that call each other (i.e., all bidirectional link) will have 6 hops `(A → B, B → A, B → C, C → B, A → C, C → A)`

We can’t calculate the amount of hops automatically based upon the nodes,
but it should be a value between `#services - 1` and `#services!`.

If we know the amount of hops in a system, we can calculate the cardinality of the generated
[service graphs]({{< relref "../service_graphs" >}}) (assuming `#hb` is the number of histogram buckets):

```
  traces_service_graph_request_total: #hops
  traces_service_graph_request_failed_total: #hops
  traces_service_graph_request_server_seconds: #hb * #hops
  traces_service_graph_request_client_seconds: #hb * #hops
  traces_service_graph_unpaired_spans_total: #services (absolute worst case)
  traces_service_graph_dropped_spans_total: #services (absolute worst case)
```

Finally, we get the following cardinality estimation:

```
  Sum: [([2 * #hb] + 2) * #hops] + [2 * #services]
```

{{< admonition type="note" >}}
If `enable_messaging_system_latency_histogram` configuration is set to `true`, another histogram will be produced:

```
  traces_service_graph_request_messaging_system_seconds: #hb * #hops
```

In that case, the estimation formula would be:

```
  Sum: [([3 * #hb] + 2) * #hops] + [2 * #services]
```

{{% /admonition %}}

{{< admonition type="note" >}}
To estimate the number of metrics, refer to the [Dry run metrics generator]({{< relref "../cardinality" >}}) documentation.
{{% /admonition %}}