---
Authors: Koenraad Verheyden (@kvrhdn), Mario Rodriguez (@mapno)
Created: December 2021 - January 2022
Last updated: 2022-01-xx
---

# Metrics-generator

## Summary

This design document describes adding a mechanism to Tempo that can generate metrics from ingested spans.

To generate metrics we propose adding a new optional component: the _metrics-generator_. If present, the distributor will write received spans to both the ingester and the metrics-generator. The metrics-generator processes the spans and writes metrics to a Prometheus datasource using the Prometheus remote write protocol.

_Note: this feature is sometimes referred to as "server-side metrics". The Grafana Agent already supports these capabilities (to generate metrics from traces), in that context moving these processors from the Agent into Tempo moves them server-side._

## Architecture and implementation

Generating and writing metrics adds a whole new feature to Tempo unlike any other functionality thus far. Instead of integrating this into an existing component, we propose adding a new component dedicated to this work. This results in a clean division of responsibility and limits the blast radius from a metrics processors or the Prometheus remote write exporter blowing up.

Alternatives considered:

- integrate into the distributor: as some processors (i.e. service graphs) have to process all spans of a trace, this would either require trace-aware load balancing to the distributor or an external store shared by all instances. This would complicate the deployment of the distributor and distract from its main responsibility.

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

The rest of this section describes the different components in more detail, ordered by the flow of the data.

### Distributor

After writing data to the ingesters, the distributor will also write to the metrics-generator (if enabled). Writing to the metrics-generator is on a _best effort basis_: even if writing to the generator fails the distributor write is still considered successful. This is a trade-off to keep request handling simple: if writing to the ingester succeeds but writing to the metrics-generator fails, the distributor should also revert the ingester write. The logic to discard already ingested data will be too complex.

This tradeoff will result in missing or incomplete metrics whenever the metrics-generator is not able to ingest some data. As the client will not be aware of this, it will not resend the request. Failed writes should be reported with a metric on the distributor which can alert an operator (e.g. `distributor_metrics_generator_pushes_failures_total`).

#### Metrics-generator ring

The distributor has to detect metrics-generator instances present in the cluster. When multiple instances of the metrics-generator are running, the distributor should load balance writes across these instances. Load balancing has to be trace-aware: all spans with the same trace ID should consistently be sent to the same instance.

To achieve this we propose using the [dskit ring](https://github.com/grafana/dskit/tree/main/ring) backed by memberlist. This will be the same mechanism as used by the ingesters.

In a later revision we can look into running the metrics-generators with a replication factor of 2 or higher. This is already supported by the dskit ring, but will require extra logic to deduplicate metrics when exporting them (otherwise they are counted multiple times).  
This is out-of-scope for this design document. Initially the metrics-generator will run with RF=1 which will result in data loss when an instance crashes.

#### gRPC protocol

Similar to other Tempo components, inter-component requests are sent over gRPC. The existing APIs are defined in [`tempopb/tempo.proto`](../../pkg/tempopb/tempo.proto).

Service definition:

```protobuf
service MetricsGenerator {
  rpc PushSpans(PushSpansRequest) returns (PushResponse) {};
}

// Note: a PushSpansRequest should only contain spans that are relevant to the configured processors
// and tenants. We can reduce the amount of data sent by aggressively trimming spans and their
// metadata in the distributor. 
message PushSpansRequest {
  // Note: these are full traces. For further optimisation we should consider using a slimmer span
  // format containing the minimal amount of data.
  repeated tempopb.trace.v1.ResourceSpans batches = 1;
}

message PushResponse {
}
```

Since the metrics-generator is directly in the write path, an increase in ingress traffic will directly impact the metrics-generator. To reduce the amount of data sent from the distributor to the metrics-generator, the distributor should only send spans that are relevant for the configured metrics processors and tenants. If, for example, a processor only requires a subset of spans the distributor should drop not relevant spans. This should allow the metrics-generator to scale slower than the ingesters.

### Metrics-generator

Diagram of what the metrics-generator could look like internally:

```
           Metrics-generator
          ┌──────────────────────────────────────────────────────────────────────────────────┐
          │              1 instance per tenant                                               │
          │             ┌ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─                    │
          │                                                              │                   │
          │             │                                                                    │
          │                      ┌──────────────┐                        │                   │
          │             │ Spans  │   Metrics    │ Metrics                                    │
          │                ┌────▶│ processor #1 │───┐                    │                   │
          │             │  │     └──────────────┘   │                                        │
          │ ┌────────┐     │     ┌──────────────┐   │    ┌────────────┐  │   ┌────────────┐  │                  ┌ ─ ─ ─ ─ ─ ─ ─ ─
          │ │  gRPC  │  │  │     │   Metrics    │   │    │prometheus. │      │Remote write│  │                                   │
───Spans──┼▶│ Server │─────┼────▶│ processor #2 │───┼───▶│  Appender  │──┼──▶│   client   │──┼───Prometheus ───▶│   Prometheus
          │ └────────┘  │  │     └──────────────┘   │    └────────────┘      └────────────┘  │  Remote Write                     │
          │                │                        │                    │                   │                  └ ─ ─ ─ ─ ─ ─ ─ ─
          │             │  │                        │                                        │
          │                └────▶      ...       ───┘                    │                   │
          │             │                                                                    │
          │              ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ┘                   │
          └──────────────────────────────────────────────────────────────────────────────────┘
```

#### Metrics processor

Processors run inside the metrics-generator, they ingest span batches and export metrics. If Tempo is run in multi-tenant mode, the processors will run independently for every tenant and each tenant should have their own configuration. It should be possible to (re)configure processors at run-time, without restarting any components.

Processes might build up some state as parts of a trace are received. This state will be kept in-memory and will be lost if the metrics-generator restarts.

Which processors will be included in an initial implementation are described in [Initial processors](#initial-processors).

#### Metrics processor configuration

As configuration has to be reloaded at run-time. This mechanism already exists in Tempo: the overrides. An initial implementation of the metrics-generator should allow configuring processors for every tenant using the overrides configuration file.

Out-of-scope: in a later revision we can consider adding a management API to configure the processors for a tenant. This would allow tenants to self-serve metrics. Before this can be implemented, limits should be in place to protect both the Tempo cluster and the metrics database against excessive metrics or high cardinality. The architecture would be like:

```
┌────────────┐                 ┌──────────────────────┐                     ┌────────────┐
│   Client   │─────Manage ────▶│  Metrics-generator   │────Store/fetch ────▶│   Bucket   │
└────────────┘   processors    └──────────────────────┘      config         └────────────┘
```

#### Prometheus remote write

The metrics exported by the metrics processors are buffered in the `prometheus.Appender` and exported using the Prometheus remote write protocol.

It's possible to configure a Prometheus remote write-compatible endpoint in the metrics-generator. When Tempo is run in multi-tenant mode, the `X-Scope-OrgID` header used to identify a tenant will be forward to the Prometheus-compatible backend.

Out-of-scope: in a later revision we can consider adding a WAL to persist metrics.

## Initial processors

_Note: these processors also exist in the Grafana Agent. Ideally the metrics exported by Tempo are the same as the metrics from the Agent so a frontend (e.g. Grafana) can work with both._

### Service graphs

Information about service graphs generated by the Grafana Agent: https://grafana.com/docs/tempo/next/grafana-agent/service-graphs/

TODO describe what metrics will be exported and how they are built up.

### Span metrics

Information about span metrics generated by the Grafana Agent: https://grafana.com/docs/tempo/next/grafana-agent/span-metrics/

TODO describe what metrics will be exported and how they are built up.

## Configuration

> Note: this section is still very preliminary and will change during the design process.

Example of what the configuration of the distributor and the metrics-generator could look like:

```yaml
distributor:
  # TODO distributor has to know somehow it should enable the metrics-generator ring
  metrics_generator_present: true

# Similar to the ingester_client, configure the client used by the distributor
metrics_generator_client:
  # Same settings as ingester_client

metrics_generator:
  # TODO global resolution / interval settings? e.g. DPM settings

  remote_write:
    enabled: true
    # prometheus.RemoteWriteClient
    client:
      url: http://prometheus:9090/prometheus/api/v1/writ
```

Example of what the overrides could look like:

```yaml
overrides:
  1:
    # TODO configure processors here
  2:
    # TODO configure some other processors here
```

TODO figure out how to set limits

## Production readiness and more

This section lists work that should be done to make metrics-generator a production-ready feature, note this list is not exhaustive.

- Add the ability to run with RF=3.
- Look into data loss incurred during a crash. Maybe investigate if a WAL could solve this, though we will have to persist both the internal state of the processors and the generated metrics thus far.