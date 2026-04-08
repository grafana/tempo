---
title: Distributor
menuTitle: Distributor
description: How the distributor receives, validates, and routes trace data.
weight: 100
topicType: concept
versionDate: 2026-03-20
---

# Distributor

The distributor is the entry point for all trace data into Tempo.
It receives spans from instrumented applications, validates them against configured limits, and writes them to Kafka.

## Receiving traces

The distributor uses the receiver layer from the [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector) and accepts spans in multiple formats:

- OpenTelemetry Protocol (OTLP) over gRPC and HTTP (recommended)
- Jaeger (Thrift and gRPC)
- Zipkin
- Kafka

We recommend using OTLP over gRPC when possible.
Both [Grafana Alloy](https://github.com/grafana/alloy/) and the OpenTelemetry Collector support OTLP export natively.

## Validation and rate limiting

Before writing to Kafka, the distributor validates incoming data against configured ingestion limits.
These are the only limits enforced synchronously at ingestion time.

The ingestion rate limit sets the maximum bytes per second per tenant.
Exceeding this returns a `RATE_LIMITED` error to the client.
The ingestion burst size controls the maximum burst allowed above the sustained rate.

Other limits such as `max_bytes_per_trace` and `max_live_traces_bytes` are enforced asynchronously downstream by live-stores and block-builders.

When the distributor refuses spans due to rate limits,
it increments the `tempo_discarded_spans_total` metric with a `reason` label indicating why.

### Logging discarded spans

To log individual discarded spans for debugging:

```yaml
distributor:
  log_discarded_spans:
    enabled: true
    include_all_attributes: false
```

Setting `include_all_attributes: true` produces more verbose logs that include span attributes,
which can help identify misbehaving clients.

## Writing to Kafka

After validation, the distributor shards traces by hashing the trace ID,
looks up the partition ring to determine which Kafka partitions are active,
and writes records to the appropriate partitions.
It waits for Kafka to acknowledge the write before returning a response to the client.

The write is only considered successful after Kafka returns with success.
This ensures that once the client gets a success response, the data is durably stored.

### Partitioning

The distributor shards traces by trace ID, meaning all spans for the same trace go to the same Kafka partition.
This has two benefits:

- Block-builders can build blocks where all spans for a trace are co-located (within a single consumption cycle).
- Live-stores can serve complete traces from a single partition without cross-partition coordination.

The distributor uses the partition ring (not Kafka's partition routing) to determine target partitions.
This allows Tempo to control the partition lifecycle independently of Kafka.

## Key metrics

| Metric | Description |
|---|---|
| `tempo_distributor_spans_received_total` | Total spans received by the distributor |
| `tempo_discarded_spans_total` | Spans discarded, labeled by `reason` |
| `tempo_distributor_bytes_received_total` | Total bytes received |
| `tempo_distributor_ingestion_rate_spans` | Current ingestion rate in spans per second |

## Related resources

Refer to the [distributor configuration](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#distributor) for the full list of options.
