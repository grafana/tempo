# Tempo

Tempo is a Jaeger/Zipkin/OpenCensus compatible backend.  It is not OpenTelemetry compatible only b/c that doesn't exist yet.  Tempo ingests batches in any of the mentioned formats, buffers them and then writes them to GCS.

## Getting Started

See the [example folder](example) for various ways to get started running tempo locally.

## Architecture

Tempo is built around the Cortex architecture.  It vendors Cortex primarily for the ring/lifecycler code. 

### distributor

Distributors vendor the OpenTelemetry Collector to reuse their [receiver code](https://github.com/grafana/tempo/tree/master/pkg/distributor/receiver) and then use consistent ring hashing to split up a batch and push it to ingesters based on trace id.

### ingester

Ingesters batch traces until a configurable timeout is hit and then push them into a headblock.  Blocks are cut periodically and shipped to the backend (gcs).

### querier

Queriers request trace ids both from ingesters and the backend and return the set of batches matching the requested trace id.

### compactor

Compactors iterate over all blocks looking for candidates for compaction.  They are scaleable and use a consistent ring to decide ownership of a given set of blocks.

## Other Components

### tempo-query
tempo-query is jaeger-query with a [hashicorp go-plugin](https://github.com/jaegertracing/jaeger/tree/master/plugin/storage/grpc) to support querying tempo.

### tempo-vulture
tempo-vulture is tempo's bird based consistency checking tool.  It queries Loki, extracts trace ids and then queries tempo.  It metrics 404s and traces with missing spans.

### tempo-cli
tempo-cli is place to put any utility functionality related to tempo.  Currently it only supports dumping header information for all blocks from gcs.
```
go run ./cmd/tempo-cli -gcs-bucket ops-tools-tracing-ops -tenant-id single-tenant
```

## TempoDB

[TempoDB](https://github.com/grafana/tempo/tree/master/tempodb) is contained in the tempo repository but is meant to be a stand alone key value database built on top of cloud object storage (gcs/s3).

## Todo

If you are getting into the project it would be worth reviewing the list of issues to get a feel for existing work on Tempo.  Below are some of the most important issues/features to resolve before considering Tempo Beta.

- Determine and fix the reason for partial traces
  - https://github.com/grafana/tempo/issues/119
- Provide a "meta" configuration layer that tightens up config and protects Tempo from upstream changes in Cortex config.  This also includes the decision about whether or not Tempo should support ring storage mechanisms besides gossip.
  - https://github.com/grafana/tempo/issues/7
- Organize data storage around a page and implement a page aligned cache.
  - https://github.com/grafana/tempo/issues/32
  - https://github.com/grafana/tempo/issues/30
- Clean up bad blocks with the compactor.  This is importtant b/c otherwise bad blocks live forever.
   - https://github.com/grafana/tempo/issues/112
- Update otelcol dependency and move to otel proto.  This should come with significant performance gains.
  - https://github.com/grafana/tempo/issues/8
- Add a code of conduct, contributing guidelines and changelog.

And then, in order to offer hosted Tempo we would need to work out integration with Grafana.com APIs, authorization, and other things I'm not thinking of.