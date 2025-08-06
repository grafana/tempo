---
title: Tempo architecture
description: Learn about Tempo architectural decisions and operational implications.
aliases:
  - ../architecture # https://grafana.com/docs/tempo/<TEMPO_VERSION>/architecture/
  - ../operations/architecture/ # /docs/tempo/latest/operations/architecture/
weight: 400
---

# Tempo architecture

The journey of a trace is usually from an instrumented application, to a trace receiver/collector which can process the incoming trace spans before batching them up and sending them to a trace backend store (such as Tempo), and then those traces being retrieved and visualized in a tool like Grafana.

Tempo acts in two major capacities:

- An ingester of trace span, sorting the span resources and attributes into columns in an Apache Parquet schema, before sending them to object storage for long-term retention.
- Retrieving trace data from storage, either by specific trace ID or by search parameters via TraceQL.

Refer to the [example setups](https://grafana.com/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/example-demo-app/)
or [deployment options](https://grafana.com/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/deploy/) for help deploying.

![Tempo architecture diagram](/media/docs/tempo/tempo_arch.png)

## Components

Tempo is made up of the following components.

### Distributor

The distributor accepts spans in multiple formats including Jaeger, OpenTelemetry, Zipkin. It routes spans to ingesters by hashing the `traceID` and using a [distributed consistent hash ring](http://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/manage-advanced-systems/consistent_hash_ring/).
The distributor uses the receiver layer from the [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector).
For best performance, it's recommended to ingest [OTel Proto](https://github.com/open-telemetry/opentelemetry-proto).
For this reason, [Grafana Alloy](https://github.com/grafana/alloy/) uses the OTLP exporter/receiver to send spans to Tempo.

Once data is received, the Distributor determines which Ingesters to send the data to if there is more than one Ingester configured.

### Ingester

The Ingester is responsible for both indexing the received span data, and storing it into object storage.
It does so by examining the incoming span data and partitioning its span and resource attributes into the Parquet schema to allow fast retrieval later.
Ingesters build blocks of trace data for a configured period/maximum block size, before writing it to storage.
There are configurable redundancy mechanisms in Ingesters to allow for partial outage of Tempo in scaled deployments (but not in monolithic mode).

The Ingester batches trace into blocks, creates bloom filters and indexes, and then flushes it all to the backend.
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

Refer to the [Ingester](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#ingester) documentation for more information on the configuration options.

### Query Frontend

The Query Frontend is the component called by, for example, Grafana when a user wishes to retrieve a specific trace via a trace ID, or carry out a search via a TraceQL filter (for example, all incoming requests for traces).
The Query Frontend is responsible for calling one or more Queriers to carry out examination of potential blocks of data where span data for matching traces may exist in parallel, to speed up result response time (dependent on multiple Queriers being configured).
Requests by the Query Frontend can be split across multiple Queriers, which all work in parallel to retrieve results quickly.
The more Queriers available, generally the quicker a result response.
When Queriers have returned enough responses, the Query Frontend is responsible for concatenating all of the span data returned by each individual Querier together to send a response to the requester.

The Query Frontend is responsible for sharding the search space for an incoming query.

A simple HTTP endpoint exposes traces:
`GET /api/traces/<traceID>`

Internally, the Query Frontend splits the blockID space into a configurable number of shards and queues these requests.
Queriers connect to the Query Frontend via a streaming gRPC connection to process these sharded queries.

### Querier

Queriers carry out the actual examination of block data in object storage to find any relevant span data based on the trace ID or TraceQL query given.
As well as object storage, they also query Ingesters for any recent span data that may have been processed but not yet sent to object storage.
This ensures that both historical and recent trace data is always available.

The querier finds the requested trace ID in either the ingesters or the backend storage.
Depending on parameters, the querier queries the ingesters for recently ingested traces and pulls the bloom filters and indexes from the backend storage to efficiently locate the traces within object storage blocks.

The querier exposes an HTTP endpoint at:
`GET /querier/api/traces/<traceID>`, but it's not intended for direct use.

Queries should be sent to the Query Frontend.

### Compactor

The Compactor is responsible for ensuring that the stored data is both compressed and deduplicated (more on deduplication in the advanced course).
The compactor is also responsible for expiring data after the retention period for that data has been reached.
Compactors run on scheduled frequent intervals to deal with data that is not compacted and has been stored by the Ingesters.
Compaction takes into account the data stored for specific traces to minimize search space on queries.

### Object storage

Tempo uses object storage for storing all tracing data. It supports three major object storage APIs:

- Amazon Simple Storage Service (S3)
- Google Cloud Storage (GCS)
- Microsoft Azure Storage (AS)

### Metrics generator

This is an **optional** component that derives metrics from ingested traces and writes them to a metrics storage. Refer to the [metrics-generator documentation](https://grafana.com/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/metrics-generator/) to learn more.

## Things to keep in mind

There are three fundamentally important things to keep in mind with traces:

- There is no concept of the 'end' of a trace. A trace can start with any span which holds a unique trace ID that hasn't been seen by Tempo previously. However, spans can be continually added at any point in the future.

- When a trace ID is queried in Tempo, it returns all of the currently stored/ingested spans that belong to that trace and present them in a mapped response (that is, a graph of parent/child/sibling relationships between spans) that allow, for example, Grafana to then visually render those traces.

- TraceQL queries return all matching traces from Tempo for the filters presented to it. This does not necessarily mean in chronological order, as some Queriers may return results more promptly than others. When the max trace limit for a TraceQL query has been hit, the Query Frontend will return a response with all the current traces and their spans at that point and ignore/cancel all outstanding Queriers.
