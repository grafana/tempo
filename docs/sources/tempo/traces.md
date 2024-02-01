---
aliases:
  - /docs/tempo/getting-started/traces
  - /docs/tempo/traces
description: "What are traces?"
keywords:
  - Grafana
  - traces
  - tracing
title: What are traces?
weight: 120
---

# What are traces?

A trace represents the whole journey of a request or an action as it moves through all the nodes of a distributed system, especially containerized applications or microservices architectures.
Traces let you profile and observe systems, making it easy to discover bottlenecks and interconnection issues.

Traces are composed of one or more spans.
A span is a unit of work within a trace that has a start time relative to the beginning of the trace, a duration and an operation name for the unit of work. It usually has a reference to a parent span (unless it is the first span in a trace).
It may optionally include key/value attributes that are relevant to the span itself, for example the HTTP method being used by the span, as well as other metadata such as sub-span events or links to other spans.

By definition, traces are never complete. You can always push a new batch of spans, even if days have passed since the last one.
When receiving a query requesting a stored trace, tracing backends (for example Tempo), find all the spans for that specific trace and collate them into a returned result. For that reason, issues can start to arise predominantly on retrieval of the trace data if you are creating traces that are extremely large in size.

<!-- Explanation of traces -->
{{< youtube id="ZirbR0ZJIOs" >}}

## Example of traces

Firstly, a user on your website enters their email address into a form to sign up for your mailing list. They click **Enter**. This initial transaction has a trace ID that is subsequently associated with every interaction in the chain of processes within a system.

Next, the user's email address is data that flows through your system. In a cloud computing world, it is possible that clicking that one button causes data to touch multiple nodes across your cluster of microservices.

As a result, the email address might be sent to a verification algorithm sitting in a microservice that exists solely for that purpose. If it passes the check, the information is stored in a database.

Along the way, an anonymization node strips personally identifying data from the address and sends metadata collected to a marketing qualifying algorithm to determine whether the request was sent from a targeted part of the internet.

Services respond and data flows back from each, sometimes triggering new events across the system. Along the way, logs are written in various nodes with a time stamp showing when the info passed through.

Finally, the request and response activity ends and a record of that request is sent to Grafana or Grafana Cloud.

## Traces and trace IDs

Setting up tracing adds an identifier, or trace ID, to all of these events. The trace ID is generated when the request is initiated and that same trace ID is applied to every single event as the request and response generate activity across the system.

That trace ID enables one to trace, or follow a request as it flows from node to node, service to microservice to lambda function to wherever it goes in your chaotic, cloud computing system and back again. This is recorded and displayed as spans.

Here's an example showing two pages in Grafana Cloud. The first, on the left (1), shows a query using the **Explore** feature. In the query results you can see a **traceID** field that was added to an application. That field contains a **Tempo** trace ID. The second page, on the right (2), uses the same Explore feature to perform a Tempo search using that **trace ID**. It then shows a set of spans as horizontal bars, each bar denoting a different part of the system.

![Traces example with query results and spans](/static/img/docs/tempo/screenshot-trace-explore-spans-g10.png)

The trace ID is applied to activities recorded as metrics and as logs.

## What are traces used for?

Traces can help you find bottlenecks.
A trace can be visualized to give a graphic representation of how long it takes for each step in the data flow pathway to complete.
It can show where new requests are initiated and end, and how your system responds.
This data helps you locate problem areas, often in places you never would have anticipated or found without this ability to trace the request flow.

Metrics, logs, and traces form the three pillars of observability.
Metrics provide the health data about the state of a system.
Logs provide an audit trail of activity that create an informational context. Traces tell you what happens at each step or action in a data pathway.

## Traces versus metrics and logs

Each observability signal plays a unique role in providing insights into your systems.
Metrics act as the high-level indicators of system health.
They alert you that something is wrong or deviating from the norm.
Logs then help you understand what exactly is going wrong, for example, the nature or cause of the elevated error rates you're seeing in your metrics.
Traces illustrate where in the sequence of events something is going wrong.
They let you pinpoint which service in the many services that any given request traverses is the source of the delay or the error.

Let's say a server takes too long to send data. Your metrics that track the latency of your system will increase, and they may then trigger an alert once that latency rises outside of an acceptable threshold.

Sending that data likely requires that a request interact with many different services in your system. Traces help you pinpoint the specific service that's introducing the added latency that you're seeing in your metrics. Alternatively, if you're seeing an elevated rate of errors when sending data, traces help you figure out from which service the errors are originating from.

Logs provide a granular view of what exactly is going wrong. For example, there could be multiple connection refused errors in your log lines. This explains why the email server took too long to send data.

<!-- What traces provide that logs and metrics don't -->
{{< youtube id="0tlp7QCPu0k" >}}

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

## Terminology

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
