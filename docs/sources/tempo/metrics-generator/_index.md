---
aliases:
- /docs/tempo/latest/server_side_metrics/
- /docs/tempo/latest/metrics-generator/
title: Metrics-generator
weight: 400
---

# Metrics-generator

Metrics-generator is an optional Tempo component that derives metrics from ingested traces.
If present, the distributor will write received spans to both the ingester and the metrics-generator.
The metrics-generator processes spans and writes metrics to a Prometheus data source using the Prometheus remote write protocol.

>**Note**: Enabling metrics generation and remote writing them to Grafana Cloud Metrics produces extra active series that could impact your billing. For more information on billing, refer to [Billing and usage](https://grafana.com/docs/grafana-cloud/billing-and-usage/).

## Overview

Metrics-generator leverages the data available in Tempo's ingest path to provide additional value by generating metrics from traces.

The metrics-generator internally runs a set of **processors**.
Each processor ingests spans and produces metrics.
Every processor derives different metrics. Currently the following processors are available:

- Service graphs
- Span metrics

<p align="center"><img src="server-side-metrics-arch-overview.png" alt="Service metrics architecture"></p>

### Service graphs

Service graphs are the representations of the relationships between services within a distributed system.

This service graphs processor builds a map of services by analyzing traces, with the objective to find _edges_.
Edges are spans with a parent-child relationship, that represent a jump (e.g. a request) between two services.
The amount of request and their duration are recorded as metrics, which are used to represent the graph.

To learn more about this processor, read the [documentation]({{< relref "service_graphs/" >}}).

### Span metrics

The span metrics processor derives RED (Request, Error and Duration) metrics from spans.

The span metrics processor will compute the total count and the duration of spans for every unique combination of dimensions.
Dimensions can be the service name, the operation, the span kind, the status code and any tag or attribute present in the span.
The more dimensions are enabled, the higher the cardinality of the generated metrics.

To learn more about this processor, read the [documentation]({{< relref "span_metrics/" >}}).

### Remote writing metrics

The metrics-generator runs a Prometheus Agent that periodically sends metrics to a `remote_write` endpoint.
The `remote_write` endpoint is configurable and can be any [Prometheus-compatible endpoint](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#remote_write).
To learn more about the endpoint configuration, refer to the [Metrics-generator]({{< relref "../configuration/#metrics-generator" >}}) section of the Tempo Configuration documentation.
Writing interval can be controlled via `metrics_generator.registry.collection_interval`.

When multi-tenancy is enabled, the metrics-generator forwards the `X-Scope-OrgID` header of the original request to the remote_write endpoint.
