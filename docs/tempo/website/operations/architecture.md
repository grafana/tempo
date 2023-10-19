---
aliases:
- /docs/tempo/v1.2.1/operations/architecture/
- /docs/tempo/v1.2.x/architecture/
- /docs/tempo/v1.2.x/architecture/architecture/
title: Architecture
weight: 1
---

A collection of documents that detail Tempo architectural decisions and operational implications.

# Architecture

This topic provides an overview of the major components of Tempo.  Refer to the [example setups]({{< relref "../getting-started/example-demo-app" >}})
or [deployment options]({{< relref "./deployment" >}}) for help deploying.

<p align="center"><img src="../tempo_arch.png" alt="Tempo Architecture"></p>

## Tempo
Tempo comprises of the following top-level components.

### Distributor

The distributor accepts spans in multiple formats including Jaeger, OpenTelemetry, Zipkin. It routes spans to ingesters by hashing the `traceID` and using a [distributed consistent hash ring]({{< relref "consistent_hash_ring" >}}).

The distributor uses the receiver layer from the [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector).
For best performance it is recommended to ingest [OTel Proto](https://github.com/open-telemetry/opentelemetry-proto).  For this reason
the [Grafana Agent](https://github.com/grafana/agent) uses the otlp exporter/receiver to send spans to Tempo.

### Ingester

The Ingester batches trace into blocks, creates bloom filters and indexes, and then flushes it all to the backend. 
Blocks in the backend are generated in the following layout.

```
<bucketname> / <tenantID> / <blockID> / <meta.json>
                                      / <index>
                                      / <data>
                                      / <bloom_0>
                                      / <bloom_1>
                                        ...
                                      / <bloom_n>
```

### Query Frontend

The Query Frontend is responsible for sharding the search space for an incoming query.

Traces are exposed via a simple HTTP endpoint:
`GET /api/traces/<traceID>`

Internally, the Query Frontend splits the blockID space into a configurable number of shards and queues these requests.
Queriers connect to the Query Frontend via a streaming gRPC connection to process these sharded queries.

### Querier

The querier is responsible for finding the requested trace id in either the ingesters or the backend storage. Depending on
parameters it will query both the ingesters and pull bloom/indexes from the backend to search blocks in object
storage.

The querier exposes an HTTP endpoint at:
`GET /querier/api/traces/<traceID>`, but its not expected to be used directly.

Queries should be sent to the Query Frontend.

### Compactor

The Compactors stream blocks to and from the backend storage to reduce the total number of blocks.

### Using older versions of Grafana

When using older versions of Grafana (7.4.x), you must also use `tempo-query` in order to visualize traces. The
`tempo-query` is [Jaeger Query](https://www.jaegertracing.io/docs/1.19/deployment/#query-service--ui) with a [GRPC Plugin](https://github.com/jaegertracing/jaeger/tree/master/plugin/storage/grpc) that allows it to query Tempo.
For more information, refer to [these example](https://github.com/grafana/tempo/tree/main/example/docker-compose/grafana7.4) and [these docs](../../configuration/querying).
