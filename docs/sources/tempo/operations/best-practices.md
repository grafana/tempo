---
title: Best practices for traces
menuTitle: Best practices
description: Learn about the best practices for traces
weight: 20
---

# Best practices for traces

This page provides some general best practices for tracing.

## Span and resource attributes

[Traces]({{< relref "../traces" >}}) are built from spans, which denote units of work such as a call to, or from, an upstream service. Spans are constructed primarily of span and resource attributes.
Spans also have a hierarchy, where parent spans can have children or siblings.

In the screenshot below, the left side of the screen (1) shows the list of results for the query. The right side (2) lists each span that makes up the selected trace.

![Traces example with query results and spans](/static/img/docs/tempo/screenshot-trace-explore-spans-g10.png)

A **span attribute** is a key/value pair that provides context for its span. For example, if the span deals with calling another service via HTTP, an attribute could include the HTTP URL (maybe as the span attribute key `http.url`) and the HTTP status code returned (as the span attribute `http.status_code`). Span attributes can consist of varying, non-null types.

Unlike a span attribute, a **resource attribute** is a key/value pair that describes the context of how the span was collected. Generally, these attributes describe the process that created the span.
This could be a set of resource attributes concerning a Kubernetes cluster, in which case you may see resource attributes, for example: `k8s.namespace`, `k8s.container_name`, and `k8s.cluster`.
These can also include information on the libraries that were used to instrument the spans for a trace, or any other infrastructure information.

For more information, read the [Attribute and Resource](https://opentelemetry.io/docs/specs/otel/overview/) sections in the OpenTelemetry specification.

### Naming conventions for span and resource attributes

The OpenTelemetry project defines a number of semantic conventions for attributes, which can help you to determine which attributes are most important to include in your spans. These conventions provide a common vocabulary for describing different types of entities, which can help to ensure that your data is consistent and meaningful.

When naming attributes, use consistent, nested namespaces to ensures that attribute keys are obvious to anyone observing the spans of a trace and that common attributes can be shared by spans.
Using our example from above, the `http` prefix of the attribute is the namespace, and `url` and `status_code` are keys within that namespace.
Attributes can also be nested, for example `http.url.protocol` might be `HTTP` or `HTTPS`, whereas `http.url.path` might be `/api/v1/query`.

For more details around semantic naming conventions, refer to the [Recommendations for OpenTelemetry Authors](https://opentelemetry.io/docs/specs/otel/common/attribute-naming/#recommendations-for-opentelemetry-authors) and [OpenTelemetry Semantic Conventions](https://github.com/open-telemetry/semantic-conventions/blob/main/docs/README.md) documentation.

Some third-party libraries provide auto-instrumentation that generate span and span attributes when included in a source base.

For more information about instrumenting your app for tracing, refer to the [Instrument for distributed tracing](/docs/tempo/latest/getting-started/instrumentation/) documentation.

### How many attributes should spans have?

The decision of how many attributes to include in your spans is up to you, there is no hard and fast rule.
Keep the number of attributes to a minimum, as each attribute adds overhead to the tracing system.

Only include attributes that are relevant to the operation that the span represents. For example, if you are tracing an HTTP request, you might include attributes such as the request method, the URL, and the response status code.

If you are unsure whether or not to include an attribute, it is always better to err on the side of caution and leave it out. You can always add additional attributes later if you need them.

In general, consider the following guidelines:

- Don't include metrics or logs as attributes in your spans.
- Don't use redundant attributes.
- When determining which attributes to add, consider an application's service flow, and execution in the context of only the current span.

The OpenTelemetry project does not specify a maximum number of attributes that a span can have. However, the default limits for the number of attributes per span is [128 entries](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/#attribute-limits), so you will have to adjust that. There are also default limits on attribute value and name character lengths.

## Determining where to add spans

When instrumenting, determine the smallest piece of work that you need to observe in a trace to be of value to ensure that you don’t over (or under) instrument.

Creating a new span for any work that has a relatively significant duration allows the observation of a trace to immediately show where significant amounts of time are spent during the processing of a request into your application or system.

For example, adding a span for a call to another services (either instrumented or not) may take an unknown amount of time to complete, and therefore being able to separate this work shows when services are taking longer than expected.

Adding a span for a piece of work that might call many other functions in a loop is a good signal of how long that loop is taking (you might add a span attribute that counts how many time the loop runs to determine if the duration is acceptable).
However, adding a span for each method or function call in that loop might not, as it might produce hundreds or thousands of worthless spans.

## Span length

While there are some (high) default limits to the length that a span (and by definition, the traces they belong to) can be, these can be adjusted by [these configurations]({{< relref "../configuration#ingestion-limits" >}}).
Traces that include a large number of spans and/or long-running spans can have an impact on the time taken to query them once stored.

For long-running spans and traces, the best way to see this impact on requests is to send a few test cases and see what the performance looks like (and evaluate the trace size).

From there, you can tweak the configuration for Tempo or determine ways to re-architect how the trace is being produced.

You can consider breaking up the spans in several ways:
- Decompose the query
   - For example, if a complex SQL query involves multiple operations (for example, uses joins, subqueries, or unions), consider creating separate spans for each significant operation.
- Improve granulation of long-running spans
     - For long-running operations, you could create a new span for every predetermined interval of execution time.
        {{% admonition type="note" %}}
        This requires time-based tracking in your application's code and is more complex to implement.
        {{% /admonition %}}
- Use span linking
     - Should data flow hit bottlenecks where further operations on that data might be batched at a later time, the use of [span links](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/overview.md#links-between-spans) can help keep traces constrained to an acceptable time range, while sharing context with other traces that work on the same data. This can also improve the readability of traces.
