---
Authors: Koenraad Verheyden (@kvrhdn), Mario Rodriguez (@mapno)
Created: December 2021 - January 2022
Last updated: 2022-02-07
---

# Metrics-generator

## Summary

This design document describes adding a mechanism to Tempo that can generate metrics from ingested spans.

To generate metrics we propose adding a new optional component: the _metrics-generator_. If present, the distributor will write received spans to both the ingester and the metrics-generator. The metrics-generator processes spans and writes metrics to a Prometheus datasource using the Prometheus remote write protocol.

_Note: this feature is sometimes referred to as "server-side metrics". The Grafana Agent already supports these capabilities (to generate metrics from traces), in that context moving these processors from the Agent into Tempo moves them server-side._

_Note: this proposal describes an initial implementation of the metrics-generator. As such, some features will be marked as out-of-scope (for now). This implementation should not be deemed fully production-ready yet._

### Table of Contents

- [Architecture](#architecture)
  -  [Comparison with the cortex and loki ruler](#comparison-with-the-cortex-and-loki-ruler)
- [Components](#components)
  - [Distributor](#distributor)
  - [Metrics-generator](#metrics-generator)
- [Metrics processors](#metrics-processors)
  - [Service graph](#service-graph)
  - [Span metrics](#span-metrics)
- [Configuration](#configuration)

## Architecture

Generating and writing metrics introduces a whole new domain to Tempo unlike any other functionality thus far. Instead of integrating this into an existing component, we propose adding a new component dedicated to working with metrics. This results in a clean division of responsibility and limits the blast radius from a metrics processors or the Prometheus remote write exporter blowing up.

Alternatives considered:

- integrate into the distributor: as some processors (i.e. service graph processor) have to process all spans of a trace, this would either require trace-aware load balancing to the distributor or an external store shared by all instances. This would complicate the deployment of the distributor and distract from its main responsibility.

- integrate into the ingester: this is mostly rejected because the ingester is already a very complicated and critical component, adding additional responsibilities will further complicate this.

Diagram of the ingress path with the new metrics-generator:

```
                                                                      │
                                                                      │
                                                                   Ingress
                                                                      │
                                                                      ▼
                                                          ┌──────────────────────┐
                                                          │                      │
                                                          │     Distributor      │
                                                          │                      │
                                                          └──────────────────────┘
                                                                    2│ │1
                                                                     │ │
                                                  ┌──────────────────┘ └────────┐
                                                  │                             │
                                                  ▼                             ▼
┌ ─ ─ ─ ─ ─ ─ ─ ─                     ┏━━━━━━━━━━━━━━━━━━━━━━┓      ┌──────────────────────┐
                 │                    ┃                      ┃      │                      │
│   Prometheus    ◀────Prometheus ────┃  Metrics-generator   ┃      │       Ingester       │◀───Queries────
                 │    Remote Write    ┃                      ┃      │                      │
└ ─ ─ ─ ─ ─ ─ ─ ─                     ┗━━━━━━━━━━━━━━━━━━━━━━┛      └──────────────────────┘
                                                                                │
                                                                                │
                                                                                │
                                                                                ▼
                                                                       ┌─────────────────┐
                                                                       │                 │
                                                                       │     Backend     │
                                                                       │                 │
                                                                       └─────────────────┘
```

### Comparison with the Cortex and Loki ruler

The metrics-generator looks similar to the ruler in Cortex and Loki: both the ruler and the metrics-generator are optional components that can generate metrics and remote write them. The main difference with the metrics-generator is that the ruler uses a query engine to query the ingesters and backend. The metrics-generator does not query any other component but instead consumes the ingress stream directly.

The Cortex and Loki ruler have a query engine powered by PromQL and LogQL respectively. Tempo does not have a query engine yet, so it's not possible yet to build a Tempo ruler. If at some point Tempo gets a query engine with similar capabilities, we can introduce a Tempo ruler and integrate it with the metrics-generator.

A couple of notable differences between the Tempo metrics-generator and the Cortex/Loki ruler:

- The metrics-generator has to consume the ingress stream. Because of this, the metrics-generator can only generate metrics about data that is being ingested. I.e. it's not possible to generate metrics from previously ingested data or to backfill metrics.

- The metrics-generator uses fixed processors, these are less flexible than a rule which can contain a user-defined custom query. On the other hand, these processors can perform calculations which can't be expressed in a query language. The processing done by the service graph processor for instance will be difficult to express in a query.

## Components

This section takes a more detailed look at the components involved in the path between ingesting traces and writing metrics.

### Distributor

The distributor is the entrypoint for Tempo writes: it will receives batches of spans and forwards them to the ingesters (using replication if enabled). With the metrics-generator in the system, the distributor will now also have to write data to the metrics-generator. The distributor will first write data to ingesters and if this was successful it will push the same data to the metrics-generator.

Writing to the metrics-generator is on a _best effort basis_: even if writing to the metrics-generator fails the Tempo write is still considered successful. This is a trade-off to keep request handling simple: if writing to the ingester succeeds but writing to the metrics-generator fails, the distributor should also revert the ingester write. The logic to discard already ingested data is deemed too complex.

This tradeoff will result in missing or incomplete metrics whenever the metrics-generator is not able to ingest some data. As the client will not be aware of this, it will not resend the request. Failed writes should be reported with a metric on the distributor which can alert an operator (e.g. `distributor_metrics_generator_pushes_failures_total`).

#### Metrics-generator ring

The distributor has to find metrics-generator instances present in the cluster. When multiple instances of the metrics-generator are running, the distributor should load balance writes across these instances. Load balancing has to be trace-aware: all spans with the same trace ID should consistently be sent to the same instance.

To achieve this we propose using the [dskit ring](https://github.com/grafana/dskit/tree/main/ring) backed by memberlist. This will be the same mechanism as used by the ingesters. The distributor will shard requests across metrics-generator instances based upon the tokens they own.

_Out-of-scope_: in a later revision we can look into running the metrics-generators with a replication factor of 2 or higher. This is already supported by the ring, but will require extra logic to deduplicate metrics when exporting them (otherwise they are counted multiple times).  
This is out-of-scope for this design document. Initially the metrics-generator will run with RF=1 only. Note this will result in data loss when an instance crashes.

#### gRPC protocol

Similar to other Tempo components, inter-component requests are sent over gRPC. The existing APIs are defined in [`tempopb/tempo.proto`](../../pkg/tempopb/tempo.proto).

Proposed service definition:

```protobuf
service MetricsGenerator {
  rpc PushSpans(PushSpansRequest) returns (PushResponse) {};
}

// Note: a PushSpansRequest should only contain spans that are relevant to the configured tenants
// and processors. We can reduce the amount of data sent to the metrics-generator by trimming spans
// and their metadata in the distributor.
message PushSpansRequest {
  // Note: these are full traces. For further optimisation we should consider using a slimmer span
  // format containing the minimal amount of data.
  repeated tempopb.trace.v1.ResourceSpans batches = 1;
}

message PushResponse {
}
```

Since the metrics-generator is directly in the write path, an increase in ingress will directly impact the metrics-generator. To reduce the amount of data sent from the distributor to the metrics-generator, the distributor should only send spans that are relevant for the configured metrics processors and tenants. If, for example, a processor only requires a subset of spans the distributor should drop not relevant spans before sending them. This should allow the metrics-generator to scale at a slower rate than the ingesters and saves bandwidth/processing time.

This will require that the distributor is aware of the tenants and processors configured in the metrics-generator. This configuration will thus have to be shared with both components.

### Metrics-generator

Diagram of what the metrics-generator could look like internally:

```
           Metrics-generator
          ┌──────────────────────────────────────────────────────────────────────────────────┐
          │                                                                                  │
          │              1 instance per tenant                                               │
          │              ┌ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ┐                     │
          │                       ┌──────────────┐                                           │
          │              │        │   Metrics    │     Collect metrics │                     │
          │                 ┌────▶│ processor #1 │◀─ ─ ─ ─ ─                                 │
          │              │  │     └──────────────┘          │          │                     │
          │  ┌────────┐     │     ┌──────────────┐                                           │
          │  │  gRPC  │  │  │     │   Metrics    │          │          │                     │
───Spans──┼─▶│ server │─────┼────▶│ processor #2 │◀─ ─ ─ ─ ─                                 │
          │  └────────┘  │  │     └──────────────┘          │          │                     │
          │                 │                                                                │
          │              │  │                               │          │                     │
          │                 └────▶      ...       ◀─ ─ ─ ─ ─                                 │
          │              │                                  │          │                     │
          │                                         ┌──────────────┐       ┌──────────────┐  │                   ┌ ─ ─ ─ ─ ─ ─ ─
          │              │                          │   Metrics    │   │   │ Remote write │  │                                  │
          │                                         │  collector   │──────▶│    client    │──┼───Prometheus ────▶│  Prometheus
          │              │                          └──────────────┘   │   └──────────────┘  │  Remote Write                    │
          │               ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─                      │                   └ ─ ─ ─ ─ ─ ─ ─
          │                                                                                  │
          └──────────────────────────────────────────────────────────────────────────────────┘
```

#### Metrics processor

Processors run inside the metrics-generator, they ingest span batches and keep track of metrics. To ensure isolation between tenants, the metrics processors are run per tenant and each tenant has their own configuration. It should be possible to (re)configure the processors without restarting the metrics-generator.

Processes might build up some state as parts of a trace are received. This state will be kept in-memory and will be lost if the metrics-generator crashes.

The implementation of the processors is described in more detail in [Metrics processors](#metrics-processors)

_Out-of-scope_: persist the state of the processors to minimize data loss during a crash.

#### Metrics processor configuration

Configuration has to be reloadable at run-time. Tempo already uses the overrides to configure limits dynamically. The metrics-generator can piggyback on this system and read per-tenant configuration from the overrides.

_Out-of-scope_: add a management API to configure the processors for a tenant. This should allow tenants to configure their own processors using a command line tool (similar to [cortextool](https://github.com/grafana/cortex-tools#cortextool)). Configuration would be written to and read from a bucket. Before this can be implemented, limits should be in place to protect both the Tempo cluster and the metrics database against excessive metrics or high cardinality. The flow of such a request would look like:

```
┌────────────┐                 ┌──────────────────────┐                     ┌────────────┐
│   Client   │─────Manage ────▶│  Metrics-generator   │────Store/fetch ────▶│   Bucket   │
└────────────┘   processors    └──────────────────────┘      config         └────────────┘
```

#### Metrics collector & Prometheus remote write

The metrics collector is a little process within the metrics-generator that on regular intervals collects metric samples from the processors. The samples are then written to a time series database using the Prometheus remote write protocol. The collector should work similar to a Prometheus instance scraping a host. So it should be possible to configure the collection interval and add external labels.

When Tempo is run in multi-tenant mode, the `X-Scope-OrgID` header used to identify a tenant will be forwarded to the Prometheus-compatible backend.

_Potential future feature_: also support writing OTLP metrics.

## Metrics processors

The metrics processors are at the core of the metrics-generator, they are responsible for converting trace data into metrics. This initial proposal describes two processors that already exist in the Grafana Agent: the service graph processor and the span metrics processor. The implementation of a processor should be flexible enough, so it's easy to add additional processors at a later stage.

### Service graph

_Note: this processor also exist in the Grafana Agent. Ideally the metrics exported by Tempo match exactly with the metrics from the Agent so a frontend (e.g. Grafana) can work with both._

A service graph is a visual representation of the interrelationships between various services. The service graph processor will analyse trace data and generate metrics describing the relationship between the services. These metrics can be used by e.g. Grafana to draw a service graph.

The goal is to mirror the implementation from the Grafana Agent. Service graphs are described [here](https://grafana.com/docs/tempo/next/grafana-agent/service-graphs/). The [Agent code lives here](https://github.com/grafana/agent/tree/release-v0.21.2/pkg/traces/servicegraphprocessor).

The service graph processor builds its metadata by analysing edges in the trace: an edge is two spans with a parent-child relationship of which the parent span has [SpanKind](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/trace/api.md#spankind) `client` and the child span has SpanKind `server`. Each edge represents a request from one service to another. The amount of requests and their duration are recorded in metrics.

The following metrics should be exported:

| Metric                                      | Type      | Labels         | Description                                                  |
|---------------------------------------------|-----------|----------------|--------------------------------------------------------------|
| traces_service_graph_request_total          | Counter   | client, server | Total count of requests between two nodes                    |
| traces_service_graph_request_failed_total   | Counter   | client, server | Total count of failed requests between two nodes             |
| traces_service_graph_request_server_seconds | Histogram | client, server | Time for a request between two nodes as seen from the server |
| traces_service_graph_request_client_seconds | Histogram | client, server | Time for a request between two nodes as seen from the client |
| traces_service_graph_unpaired_spans_total   | Counter   | client, server | Total count of unpaired spans                                |
| traces_service_graph_dropped_spans_total    | Counter   | client, server | Total count of dropped spans                                 |

Since the service graph processor has to process both sides of an edge, it needs to process all spans of a trace to function properly. If spans of a trace are spread out over multiple instances it will not be possible to pair up spans reliably.

The following aspects should be configurable:
- `success_codes`:  the status code considered a successful request, this is used to distinguish between successful and failed requests.
- `buckets`: the buckets to use for the histogram.

### Span metrics

The span metrics processor aggregates request, error and duration metrics (RED) from span data.

The goal is to mirror the implementation from the OpenTelemetry Collector. Span metrics are described [here](https://grafana.com/docs/tempo/next/grafana-agent/span-metrics/). The [code lives here](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/spanmetricsprocessor).

The span metrics processor will compute the total count and the duration of spans for every unique combination of dimensions. Dimensions can be the service name, the operation, the span kind, the status code and any tag or attribute present in the span. The more dimensions are enabled, the higher the cardinality of the generated metrics.

The following metrics should be exported:

| Metric                               | Type      | Labels     | Description             |
|--------------------------------------|-----------|------------|-------------------------|
| traces_spanmetrics_duration_seconds | Histogram | Dimensions | Duration of the span    |
| traces_spanmetrics_calls_total      | Counter   | Dimensions | Total count of the span |

The following aspects should be configurable:
- `dimensions`:  the labels to include in the generated metrics, each dimension must match with an attribute of the span.
- `buckets`: the buckets to use for the histogram.
- `include`/`exclude`: filter which spans to include when generating span metrics. It should be possible to only generate span metrics for a subet of services for instance.

## Configuration

Example of what the configuration of the distributor and the metrics-generator could look like:

_Note: this is just a proposal, the final configuration can be found in the documentation._ 

```yaml
distributor:
  # Toggle to enable or disable the metrics-generator ring. If disabled, the distributor should
  # not initialize the metrics-generator ring and does not send data to the metrics-generator.
  enable_metrics_generator_ring: true

# Similar to the ingester_client, configure the client used by the distributor
metrics_generator_client:
  # Same settings as ingester_client

metrics_generator:
  collection_interval: 15s
  external_labels:
    some_static_label: foo 

  # Global settings for the metrics processors
  processor:
    service_graphs:
      histogram_buckets: [0.1, 0.2, 0.5, 1, 2, 5, 10]
    span_metrics:
      dimensions:
        - http.method
        - http.target

  # Configure remote write target
  remote_write:
    enabled: true
    client:
      # prometheus.RemoteWriteConfig
      # https://prometheus.io/docs/prometheus/latest/configuration/configuration/#remote_write
      url: http://prometheus:9090/prometheus/api/v1/write
```

Example of what the overrides could look like:

```yaml
overrides:
  1:
    metrics_generator_processors:
      - service-graphs
      - span-metrics
  2:
    metrics_generator_processors:
      - service-graphs
```
