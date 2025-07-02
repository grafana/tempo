---
title: Tempo architecture
description: Learn about Tempo architectural decisions and operational implications.
aliases:
 - ../architecture/architecture # https://grafana.com/docs/tempo/<TEMPO_VERSION>/architecture/architecture/
 - ../architecture # https://grafana.com/docs/tempo/<TEMPO_VERSION>/architecture/
weight: 100
---

# Tempo architecture

This topic provides an overview of the major components of Tempo. Refer to the [example setups](https://grafana.com/docs/tempo/<TEMPO_VERSION>/getting-started/example-demo-app/)
or [deployment options](https://grafana.com/docs/tempo/<TEMPO_VERSION>/setup/deployment/) for help deploying.

![Tempo architecture diagram](/media/docs/tempo/tempo_arch.png)

Tempo comprises of the following top-level components.

## Distributor

The distributor accepts spans in multiple formats including Jaeger, OpenTelemetry, Zipkin. It routes spans to ingesters by hashing the `traceID` and using a [distributed consistent hash ring](http://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/manage-advanced-systems/consistent_hash_ring/).
The distributor uses the receiver layer from the [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector).
For best performance, it's recommended to ingest [OTel Proto](https://github.com/open-telemetry/opentelemetry-proto).
For this reason, [Grafana Alloy](https://github.com/grafana/alloy/) uses the OTLP exporter/receiver to send spans to Tempo.

## Ingester

The [Ingester](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#ingester) batches trace into blocks, creates bloom filters and indexes, and then flushes it all to the backend.
Blocks in the backend are generated in the following layout:

```
<bucketname> / <tenantID> / <blockID> / <meta.json>
                                      / <index>
                                      / <data>
                                      / <bloom_0>
                                      / <bloom_1>
                                        ...
                                      / <bloom_n>
```

## Query Frontend

The Query Frontend is responsible for sharding the search space for an incoming query.

A simple HTTP endpoint exposes traces:
`GET /api/traces/<traceID>`

Internally, the Query Frontend splits the blockID space into a configurable number of shards and queues these requests.
Queriers connect to the Query Frontend via a streaming gRPC connection to process these sharded queries.

## Querier

The querier finds the requested trace ID in either the ingesters or the backend storage. Depending on
parameters, the querier queries the ingesters for recently ingested traces and pulls the bloom filters and indexes from the backend storage to efficiently locate the traces within object storage blocks.

The querier exposes an HTTP endpoint at:
`GET /querier/api/traces/<traceID>`, but it's not intended for direct use.

Queries should be sent to the Query Frontend.

## Compactor

The Compactors stream blocks to and from the backend storage to reduce the total number of blocks.

## Metrics generator

This is an **optional** component that derives metrics from ingested traces and writes them to a metrics storage. Refer to the [metrics-generator documentation](https://grafana.com/docs/tempo/<TEMPO_VERSION>/metrics-generator/) to learn more.
