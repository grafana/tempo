---
title: Long-running traces
description: Troubleshoot search results when using long-running traces
weight: 479
aliases:
  - ../../operations/troubleshooting/long-running-traces/ # https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/troubleshooting/long-running-traces/
  - ../long-running-traces/ # https://grafana.com/docs/tempo/<TEMPO_VERSION>/troubleshooting/long-running-traces/
---

# Long-running traces

Long-running traces are created when Tempo receives spans for a trace,
followed by a delay, and then Tempo receives additional spans for the same
trace. If the delay between spans is great enough, the spans end up in
different blocks, which can lead to inconsistency in a few ways:

1. When using TraceQL search, the duration information only pertains to a
   subset of the blocks that contain a trace. This happens because Tempo
   consults only enough blocks to know the TraceID of the matching spans. When
   performing a TraceID lookup, Tempo searches for all parts of a trace in all
   matching blocks, which yields greater accuracy when combined.

1. When using [`spanset`
   operators](https://grafana.com/docs/tempo/<TEMPO_VERSION>/traceql/construct-traceql-queries/#combine-spansets),
   Tempo only evaluates the contiguous trace of the current block. This means
   that for a single block the conditions may evaluate to false, but to
   consider all parts of the trace from all blocks would evaluate true.


In Tempo 3.0, two components handle trace data independently:

- **Live-stores** serve recent data. They hold traces in memory and can keep spans for the same trace together as long as the trace remains active. You can tune the `live_store.max_trace_idle` configuration to control when a trace is considered idle. Extending this beyond the default `5s` can allow for long-running traces to be co-located, but take into account other considerations around memory consumption on the live-stores.
- **Block-builders** consume from Kafka and build blocks for long-term storage. They do a hard cut at a certain record on each consumption cycle. All spans consumed in a cycle are flushed into blocks regardless of whether the trace is complete. This means a trace's spans can be split across block-builder cycles with no way to keep them together.

### Data quality metrics

Tempo publishes a `tempo_warnings_total` metric from the live-store, which
can aid in understanding when this situation arises.

When a trace is flushed to the WAL in the live-store, it's marshalled in the Parquet format which makes it available for TraceQL metrics and search.
The more complete a trace is at this moment, the more accurate complex queries are.
The `disconnected_trace_flushed_to_wal` and `rootless_trace_flushed_to_wal` metrics help operators measure how reliable their trace data pipeline is.

* `disconnected_trace_flushed_to_wal`: Incremented when a trace is flushed that has a span with parent id that cannot be found.
* `rootless_trace_flushed_to_wal`: Incremented when a trace is flushed that doesn't have a root span. A root span is a span with all `0` parent id.

You might see these data quality metrics if you use a Prometheus query like this to explore Tempo warnings:

```
sum(rate(tempo_warnings_total{}[5m])) by (reason)
```

This example helps determine the percentage of complete traces flushed. This metric can help you optimize your instrumentation and traces pipeline and understand the impact it has on Tempo data quality.

In particular, the following query can be used to know what percentage of traces flushed to the WAL are connected.

```
1 - sum(rate(tempo_warnings_total{reason="disconnected_trace_flushed_to_wal"}[5m])) / sum(rate(tempo_live_store_traces_created_total{}[5m]))
```

If you have long-running traces, you may also be interested in the
`rootless_trace_flushed_to_wal` reason to know when a trace is flushed to the
WAL without a root span.

You can use `reason` fields for discovery with this query:

```
sum(rate(tempo_warnings_total{}[5m])) by (reason)
```

In general, Tempo functions at its peak when all parts of a trace are stored
within as few blocks as possible. There is a wide variety of tracing patterns
in the wild, which makes it impossible to optimize for all of them.

While the preceding information can help determine what Tempo is doing, it may
be worth modifying the usage pattern slightly. For example, you may want to use
[span
links](https://opentelemetry.io/docs/concepts/signals/traces/#span-links), so
that traces are split up, allowing one trace to complete, while pointing to the
next trace in the causal chain . This allows both traces to finish in a
shorter duration, and increase the chances of ending up in the same block.
