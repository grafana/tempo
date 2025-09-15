---
headless: true
description: Shared file for TraceQL query structure concept.
labels:
  products:
    - enterprise
    - oss
---

[//]: # 'This file explains TraceQL query structure.'
[//]: # 'This shared file is included in these locations:'
[//]: # '/grafana/docs/sources/datasources/tempo/traceql/trace-structure.md'
[//]: # '/grafana/docs/sources/datasources/tempo/introduction/trace-structure.md'
[//]: # '/explore-profiles/docs/concepts/trace-structure.md'
[//]: # '/website/docs/grafana-cloud/send-data/traces/trace-structure.md'
[//]: #
[//]: # 'If you make changes to this file, verify that the meaning and content are not changed in any place where the file is included.'
[//]: # 'Any links should be fully qualified and not relative.'

<!--  TraceQL query structure -->

The purpose of TraceQL is to search or query for spans.
The query returns a set of spans, also called a spanset.

A TraceQL query can select traces based on:

- span attributes, timing, and duration
- structural relationships between spans
- aggregated data from the spans in a trace

Refer to [Trace Structure](https://grafana.com/docs/tempo/<TEMPO_VERSION>/introduction/trace-structure/) for information about trace structure, intrinsics, and span resources and attributes.

A query is structured as a pipeline of operations (filters and aggregators).
The query expression is evaluated on one trace at a time, selecting or discarding spans from the result.
At each stage of the query pipeline, the selected spans for a trace are grouped in a spanset (set of spans).
The associated trace is also returned. The result of the query is the spansets (and their associated traces) for all the traces evaluated.

The simplest query is this one:

```
{ }
```

The curly braces encompass the select/filter conditions.
In theory, each span (and the trace it belongs to) matching those conditions is returned by the query.
In the previous example, since there are no filter conditions, all spans are matching and thus returned with their associated traces.

In practice, the query is performed against a defined time interval, relative (for example, the last 3 hours) or absolute (for example, from X date-time to Y date-time).
The query response is also limited by the number of traces (**Limit**) and spans per spanset (**Span Limit**).

![TraceQL in Grafana](/media/docs/tempo/traceql/TraceQL-in-Grafana.png)

1. TraceQL query editor
2. Query options: **Limit**, **Span Limit** and **Table Format** (Traces or Spans).
3. Trace (by Trace ID). The **Name** and **Service** columns are displaying the trace root span name and associated service.
4. Spans associated to the Trace
