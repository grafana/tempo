---
headless: true
description: Shared file for trace structure concept.
labels:
  products:
    - enterprise
    - oss
---

[//]: # 'This file explains trace structure.'
[//]: # 'This shared file is included in these locations:'
[//]: # '/grafana/docs/sources/datasources/tempo/traceql/trace-structure.md'
[//]: # '/grafana/docs/sources/datasources/tempo/introduction/trace-structure.md'
[//]: # '/explore-profiles/docs/concepts/trace-structure.md'
[//]: # '/website/docs/grafana-cloud/send-data/traces/trace-structure.md'
[//]: #
[//]: # 'If you make changes to this file, verify that the meaning and content are not changed in any place where the file is included.'
[//]: # 'Any links should be fully qualified and not relative.'

<!--  Trace structure -->

Traces are telemetry data structured as trees.
Traces are made of spans (for example, a span tree); there is a root span that can have zero to multiple branches that are called child spans.
Each child span can itself be a parent span of one or multiple child spans, and so on so forth.

![Trace_and_spans_in_tree_structure](/media/docs/tempo/traceql/trace-tree-structures-and-spans.png)

In the specific context of Tempo and [TraceQL query language](https://grafana.com/docs/tempo/<TEMPO_VERSION>/traceql/), a span has the following associated fields:

- **name**: the span name
- **duration**: difference between the end time and start time of the span
- **status**: enum: `{ok, error, unset}`. For details, refer to [OTel span status](https://opentelemetry.io/docs/concepts/signals/traces/#span-status) documentation.
- **kind**: enum: `{server, client, producer, consumer, internal, unspecified}`. For more details, refer to [OTel span kind](https://opentelemetry.io/docs/concepts/signals/traces/#span-kind) documentation.
- Attributes

The first four properties are *intrinsics*.
They are fundamental to the span.

*Attributes* are custom span metadata in the form of key-value pairs.
There are four types of attributes: span attributes, resource attributes, event attributes, and link attributes.

*Span attributes* are key-value pairs that contain metadata that you can use to annotate a span to carry information about the operation it's tracking.
For example, in an eCommerce application, if a span tracks an operation that adds an item to a user’s shopping cart, the user ID, added item ID and cart ID can be captured and attached to the span as span attributes.

A *resource attribute* represents information about an entity producing the span.
For example, a span created by a process running in a container deployed by Kubernetes could link a resource that specifies the cluster name, namespace, pod, and container name.
Resource attributes are resource-related metadata (key-value pairs) that are describing the Resource.

An *event attribute* represents a unique point in the time during the span's duration. For more information, refer to [All about span events](https://grafana.com/blog/2024/08/15/all-about-span-events-what-they-are-and-how-to-query-them/#how-to-query-span-events-with-traceql) and [Span events](https://opentelemetry.io/docs/concepts/signals/traces/#span-events) in the OTEL documentation.

A *link attribute* lets you query link data in a span.
A span link associates one span with one or more other spans that are a causal relationship. For more information on span links, refer to the [Span Links](https://opentelemetry.io/docs/concepts/signals/traces/#span-links) documentation in the Open Telemetry project.

The OpenTelemetry specification defines Semantic Attributes for Spans and for Resources.
Semantic Span Attributes are a set of naming schemes for attributes shared across languages, frameworks, and runtimes.
For more details, refer to [Trace Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/general/trace/) and [Resource Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/resource/).