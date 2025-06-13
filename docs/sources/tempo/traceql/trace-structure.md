---
title: Trace structure and TraceQL
menuTitle: Trace structure and TraceQL
description: Learn about trace structure and TraceQL queries.
weight: 200
  - /docs/tempo/latest/traceql/architecture
keywords:
  - syntax
  - TraceQL
---

# Trace structure and TraceQL

Inspired by PromQL and LogQL, TraceQL uses similar syntax and semantics.
The differences being inherent to the very nature of searching spans and traces.

To learn more about query syntax, refer to the [Construct a TraceQL query](https://grafana.com/docs/tempo/<TEMPO_VERSION>/traceql/construct-traceql-queries/) documentation.

## Trace structure

[//]: # 'Shared content for best practices for traces'
[//]: # 'This content is located in /tempo/docs/sources/shared/trace-structure.md'

{{< docs/shared source="tempo" lookup="trace-structure.md" version="<TEMPO_VERSION>" >}}

## TraceQL queries

[//]: # 'Shared content for best practices for traces'
[//]: # 'This content is located in /tempo/docs/sources/shared/trace-structure.md'

{{< docs/shared source="tempo" lookup="traceql-query-structure.md" version="<TEMPO_VERSION>" >}}
