---
title: Long-running traces
description: Troubleshoot search results when using long-running traces
weight: 479
aliases:
  - ../operations/troubleshooting/long-running-traces/
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

1. When using structural operators, the conditions may match on different
   blocks, and so results can be confusing.

You can tune the `ingester.trace_idle_period` configuration to allow for
greater control about when traces are written to a block. Extending this beyond
the default `10s` can allow for long running trace to be co-located in the same
block, but take into account other considerations around memory consumption on
the ingesters. Currently this setting isn't per-tenant, and so adjusting
affects all ingester instances.

Tempo publishes a `tempo_warnings_total` metric from several components, which
can aid in understanding when this situation arises. In particular, the following query can be used to know what percentage of traces which are flushed to the wall are connected.

```
1 - sum(rate(tempo_warnings_total{reason="disconnected_trace_flushed_to_wal"}[5m])) / sum(rate(tempo_ingester_traces_created_total{}[5m]))
```

If you have long-running traces, you may also be interested in the
`rootless_trace_flushed_to_wal` reason to know when a trace is flushed to the
wall without a root trace.

Additional `reason` fields are available for discovery with the following
query.

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
