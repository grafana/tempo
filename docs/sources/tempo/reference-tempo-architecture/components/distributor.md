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
It receives spans from instrumented applications and validates them against configured limits.

How the distributor forwards data depends on the [deployment mode](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/deployment-modes/):

- Microservices mode: The distributor shards traces by trace ID and writes them to Kafka. Downstream components including block-builders, live-stores, and metrics-generators each consume from Kafka independently.
- Monolithic mode: The distributor pushes data in-process directly to the live-store and metrics-generator. No Kafka is required.

## Receiving traces

The distributor uses the receiver layer from the [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector) and accepts spans in multiple formats:

- OpenTelemetry Protocol (OTLP) over gRPC and HTTP, which is the recommended format
- Jaeger (Thrift and gRPC)
- Zipkin
- Kafka

We recommend using OTLP over gRPC when possible.
Both [Grafana Alloy](https://github.com/grafana/alloy/) and the OpenTelemetry Collector support OTLP export natively.

## Validation and rate limiting

Before forwarding data, the distributor validates incoming data against configured ingestion limits.
These are the only limits enforced synchronously at ingestion time.

The ingestion rate limit sets the maximum bytes per second per tenant.
Exceeding this returns a `RATE_LIMITED` error to the client.
The ingestion burst size controls the maximum burst allowed above the sustained rate.
For details on which settings honor the global strategy and which are always local, refer to [Ingestion rate strategy](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#ingestion-rate-strategy).

Other limits such as `max_live_traces_bytes` are enforced asynchronously downstream by live-stores, while `max_bytes_per_trace` is enforced downstream as well, including by block-builders in microservices mode.

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

## Writing to Kafka (microservices mode)

In microservices mode, after validation, the distributor shards traces by hashing the trace ID,
looks up the partition ring to determine which Kafka partitions are active,
and writes records to the appropriate partitions.
It waits for Kafka to acknowledge the write before returning a response to the client.

The write is only considered successful after Kafka returns with success.
This ensures that once the client gets a success response, the data is durably stored.

### Partitioning

The distributor shards traces by trace ID, meaning all spans for the same trace go to the same Kafka partition.
This has two benefits:

- Block-builders can build blocks where all spans for a trace are co-located within a single consumption cycle.
- Live-stores can serve complete traces from a single partition without cross-partition coordination.

The distributor uses the partition ring, not Kafka's partition routing, to determine target partitions.
This allows Tempo to control the partition lifecycle independently of Kafka.

## In-process push (monolithic mode)

In monolithic mode, the distributor pushes trace data directly to the live-store and metrics-generator within the same process. No Kafka producer is initialized, and the distributor doesn't use the partition ring for routing. The write is acknowledged to the client after the live-store accepts the data.

## Key metrics

| Metric | Description |
|---|---|
| `tempo_distributor_spans_received_total` | Total spans received by the distributor |
| `tempo_discarded_spans_total` | Spans discarded, labeled by `reason` |
| `tempo_distributor_bytes_received_total` | Total bytes received |
| `rate(tempo_distributor_spans_received_total[5m])` | Current ingestion rate in spans per second, derived in PromQL from the received spans counter |

## Related resources

Refer to the [distributor configuration](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#distributor) for the full list of options.
