---
title: Architecture
---

## Architecture

This document provides an overview of the major components that comprise Tempo.  Please refer to [the examples](https://github.com/grafana/tempo/tree/master/example) to see some deployment options.

<p align="center"><img src="../tempo_arch.png" alt="Tempo Architecture"></p>

## Tempo

### Distributor

Accepts spans in multiple formats including Jaeger, OpenTelemetry, Zipkin.
Routes spans to ingesters by hashing the `traceID` and using a [distributed consistent hash ring]({{< relref "consistent-hash-ring" >}}).

The distributor uses the receiver layer from the [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector).
For best performance it is recommended to ingest [OTel Proto](https://github.com/open-telemetry/opentelemetry-proto).  For this reason
the [Grafana Agent](https://github.com/grafana/agent) uses the otlp exporter/receiver to send spans to Tempo.

### Ingester

Batches traces into blocks, blooms, indexes and flushes to backend.  Blocks in the backend are generated in the following layout.

```
<bucketname> / <tenantID> / <blockID> / <meta.json>
.                                     / <bloom>
.                                     / <index>
.                                     / <data>
```

### Querier

The querier is responsible for finding the requested trace id in either the ingesters or the backend storage.  It begins by querying the ingesters to see if the id is currently stored there, if not it proceeds to use the bloom and indexes to find the trace in the storage backend.

Traces are exposed via a simple HTTP endpoint:
`GET /api/traces/<traceID>`

### Compactor

Compactors stream blocks to and from the backend storage to reduce the total number of blocks.

## Tempo-Query
Tempo itself does not provide a way to visualize traces and relies on [Jaeger Query](https://www.jaegertracing.io/docs/1.19/deployment/#query-service--ui) to do so.  `tempo-query` is [Jaeger Query](https://www.jaegertracing.io/docs/1.19/deployment/#query-service--ui) with a [GRPC Plugin](https://github.com/jaegertracing/jaeger/tree/master/plugin/storage/grpc) that allows it to speak with Tempo.

Tempo Query is also the method by which Grafana queries traces.
