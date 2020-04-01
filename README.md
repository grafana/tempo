# Tempo

Tempo is a Jaeger/Zipkin/OpenCensus compatible backend.  It is not OpenTelemetry compatible only b/c that doesn't exist yet.  Tempo ingests batches in any of the mentioned formats, buffers them and then writes them to GCS.

## Getting Started

See the [example folder](./example) for various ways to get started running tempo locally.

## Architecture

### distributor

### ingester

### querier

### compactor

## Other Components

### tempo-query

### tempo-vulture

### tempo-cli

## Compaction

## Todo

If you are getting into the project it would be worth reviewing the list of issues to get a feel for existing work on Tempo.  Below are some of the most important issues/features to resolve before considering Tempo Beta

- Determine and fix the reason for partial traces
  - https://github.com/grafana/tempo/issues/119
- Provide a "meta" configuration layer that tightens up config and protects Tempo from upstream changes in Cortex config.  This also includes the decision about whether or not Tempo should support ring storage mechanisms besides gossip.
  - https://github.com/grafana/tempo/issues/7
- Organize data storage around a page and implement a page aligned cache.
  - https://github.com/grafana/tempo/issues/32
  - https://github.com/grafana/tempo/issues/30
- Clean up bad blocks with the compactor.  This is importtant b/c otherwise bad blocks live forever.
   - https://github.com/grafana/tempo/issues/112

And then, in order to offer hosted Tempo we would need to work out integration with Grafana.com APIs, authorization, and other things I'm not thinking of.