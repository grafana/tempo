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

<p align="center"><img src="getting-started/assets/trace-explore-spans.png" alt="Trace example"></p>

A **span attribute** is a key/value pair that provides context for its span. For example, if the span deals with calling another service via HTTP, an attribute could include the HTTP URL (maybe as the span attribute key `http.url`) and the HTTP status code returned (as the span attribute `http.status_code`). Span attributes can consist of varying, non-null types.

Unlike a span attribute, a **resource attribute** is a key/value pair that describes the context of how the span was collected. Generally, these attributes describe the process that created the span.
For example, this could be a set of resource attributes concerning a Kubernetes cluster, in which case you may see resource attributes, for example: `k8s.namespace`, `k8s.container_name`, and `k8s.cluster`.
These can also include information on the libraries that were used to instrument the spans for a trace, or any other infrastructure information.

For more information, read the [Attribute and Resource](https://opentelemetry.io/docs/specs/otel/overview/) sections in the OpenTelemetry specification.


### Naming conventions for span and resource attributes

When naming attributes, use consistent, nested namespaces to ensures that attribute keys are obvious to anyone observing the spans of a trace and that common attributes can be shared by spans.
Using our example from above, the `http` prefix of the attribute is the namespace, and `url` and `status_code` are keys within that namespace.
Attributes can also be nested, for example `http.url.protocol` might be `HTTP` or `HTTPS`, whereas `http.url.path` might be `/api/v1/query`.

For more details around semantic naming conventions, refer to the [Recommendations for OpenTelemetry Authors](https://opentelemetry.io/docs/specs/otel/common/attribute-naming/#recommendations-for-opentelemetry-authors) documentation.

Some third-party libraries provide auto-instrumentation that generate span and span attributes when included in a source base.

For more information about instrumenting your app for tracing, refer to the [Instrument for distributed tracing](/docs/tempo/latest/getting-started/instrumentation/) documentation.


## Determining where to add spans

When instrumenting, determine the smallest piece of work that you need to observe in a trace to be of value to ensure that you don’t over (or under) instrument.

Creating a new span for any work that has a relatively significant duration allows the observation of a trace to immediately show where significant amounts of time are spent during the processing of a request into your application or system.

For example, adding a span for a call to another services (either instrumented or not) may take an unknown amount of time to complete, and therefore being able to separate this work shows when services are taking longer than expected.

Adding a span for a piece of work that might call many other functions in a loop is a good signal of how long that loop is taking (you might add a span attribute that counts how many time the loop runs to determine if the duration is acceptable).
However, adding a span for each method or function call in that loop might not, as it might produce hundreds or thousands of worthless spans.

## Tracing versus profiling

Tracing provides an overview of tasks performed by an operation or set of work.
Profiling provides a code-level view of what was going on.
Generally, tracing is done at a much higher level specific to one transaction, and profiling is sampled over time, aggregated over many transactions.

The superpower of tracing is seeing how a thing in one program invoked another program.

The superpower of profiling is seeing function-level or line-level detail.

For example, let’s say you want to gather trace data on how long it takes to enter and start a car. The trace would contain multiple spans:

- Walking from the resident to the car
- Unlocking the car
- Adjusting the seat
- Starting the ignition

This trace data is collected every time the car is entered and started.
You can track variations between each operation that can help pinpoint when issues happen.
If the driver forgot their keys, then that would show up as an outlying longer duration span.
In this same example, profiling gives the code stack, in minute detail: get-to-car invoked step-forward, which invoked lift-foot, which invoked contract-muscle, etc.
This extra detail provides the context that informs the data provided by a trace.
