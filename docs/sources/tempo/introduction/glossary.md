---
description: 'Glossary for traces'
keywords:
  - Grafana
  - traces
  - tracing
title: Glossary
weight: 500
---

# Glossary

The following terms are often used when discussing traces.

Active series
: {{< docs/glossary "active series" >}}

Cardinality
: {{< docs/glossary "cardinality" >}}

<!-- TODO: Add child span to the website glossary (data/glossary.yaml) and replace with shortcode. -->
Child span
: A span that is nested within a parent span. Each child span represents a sub-operation or downstream call within the broader operation represented by its parent. A child span can itself be a parent of other child spans.

Data source
: {{< docs/glossary "data source" >}}

<!-- TODO: Add event attribute to the website glossary (data/glossary.yaml) and replace with shortcode. -->
Event attribute
: A key-value pair attached to a span event, which represents a unique point in time during the span's duration. For more information, refer to [Span events](https://opentelemetry.io/docs/concepts/signals/traces/#span-events) in the OpenTelemetry documentation.

Exemplar
: {{< docs/glossary "exemplar" >}}

<!-- TODO: Add intrinsics to the website glossary (data/glossary.yaml) and replace with shortcode. -->
Intrinsics
: The core, built-in fields that are fundamental to the identity and lifecycle of spans and traces. Intrinsic fields include name, duration, status, and kind. These fields are defined by the OpenTelemetry specification and are always present.

<!-- TODO: Add link attribute to the website glossary (data/glossary.yaml) and replace with shortcode. -->
Link attribute
: A key-value pair attached to a span link. A span link associates one span with one or more causally related spans. For more information, refer to [Span Links](https://opentelemetry.io/docs/concepts/signals/traces/#span-links) in the OpenTelemetry documentation.

Log
: {{< docs/glossary "log" >}}

Metric
: {{< docs/glossary "metric" >}}

<!-- TODO: Add resource attribute to the website glossary (data/glossary.yaml) and replace with shortcode. -->
Resource attribute
: A key-value pair that represents information about the entity producing a span, such as the cluster name, namespace, Pod, or container name.

<!-- TODO: Add root span to the website glossary (data/glossary.yaml) and replace with shortcode. -->
Root span
: The first span in a trace, representing the initial request or operation. A root span has no parent span and serves as the top of the span tree.

<!-- TODO: Add semantic attribute to the website glossary (data/glossary.yaml) and replace with shortcode. -->
Semantic attribute
: A standardized naming scheme for attributes shared across languages, frameworks, and runtimes, as defined by the OpenTelemetry specification. For more information, refer to [Trace Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/general/trace/) in the OpenTelemetry documentation.

Span
: {{< docs/glossary "span" >}}

<!-- TODO: Add span attribute to the website glossary (data/glossary.yaml) and replace with shortcode. -->
Span attribute
: A key-value pair that contains metadata to annotate a span with information about the operation it tracks. For example, a span tracking an "add to cart" operation might include the user ID, item ID, and cart ID as span attributes.

Spanset
: {{< docs/glossary "spanset" >}}

Trace
: {{< docs/glossary "trace" >}}
