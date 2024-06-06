---
description: "Glossary for traces"
keywords:
  - Grafana
  - traces
  - tracing
title: Glossary
weight: 500
---

# Terminology

The following terms are often used when discussing traces.

{{< glossary.inline >}}{{ (index (where site.Data.glossary "keys" "intersect" (slice (.Get 0))) 0).value | markdownify }}{{< /glossary.inline >}}

Active series
: {{< glossary.inline "active series" />}}

Cardinality
: {{< glossary.inline "cardinality" />}}

Data source
: {{< glossary.inline "data source" />}}

Exemplar
: {{< glossary.inline "exemplar" />}}

Log
: {{< glossary.inline "log" />}}

Metric
: {{< glossary.inline "metric" />}}

Span
: {{< glossary.inline "span" />}}

Trace
: {{< glossary.inline "trace" />}}
