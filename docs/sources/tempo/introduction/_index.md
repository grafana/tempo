---
aliases:
  - /docs/tempo/getting-started/traces
  - /docs/tempo/traces
  - /docs/grafana-cloud/send-data/traces/what-are-traces
description: "What are traces? Learn about traces and how you can use them."
keywords:
  - Grafana
  - traces
  - tracing
title: Introduction
weight: 120
---

# Introduction

A trace represents the whole journey of a request or an action as it moves through all the nodes of a distributed system, especially containerized applications or microservices architectures.
Traces are the ideal observability signal for discovering bottlenecks and interconnection issues.

Traces are composed of one or more spans.
A span is a unit of work within a trace that has a start time relative to the beginning of the trace, a duration, and an operation name for the unit of work.
It usually has a reference to a parent span, unless it's the first, or root, span in a trace.
It frequently includes key/value attributes that are relevant to the span itself, for example the HTTP method used in the request, as well as other metadata such as the service name, sub-span events, or links to other spans.

By definition, traces are never complete.
You can always push another batch of spans, even if days have passed since the last one.
When receiving a query requesting a stored trace, tracing backends like Tempo find all the spans for that specific trace and collate them into a returned result.
Retrieving trace data can have issues if traces are extremely large.

<!-- Explanation of traces -->
{{< youtube id="ZirbR0ZJIOs" >}}

## Example of traces

Firstly, a user on your website enters their email address into a form to sign up for your mailing list.
They click **Enter**. This initial transaction has a trace ID that's subsequently associated with every interaction in the chain of processes within the system.

Next, the user's email address is data that flows through your system.
In a cloud computing world, it's possible that clicking that one button triggers many downstream processes on various microservices operating across many different nodes in your compute infrastructure.

As a result, the email address goes to a microservice responsible for verification. If the email passes this check, then the database stores the address.

Along the way, an anonymizing microservice strips personally identifying data from the address and adds additional metadata before sending it along to a marketing qualifying microservice.
This microservice determines whether the request came from a targeted part of the internet.

Services respond and data flows back from each, sometimes triggering additional events across the system.
Along the way, nodes write logs on which those services run with a time stamp showing when the info passed through.

Finally, the request and response activity end.
No other spans append to that trace ID.

## Traces and trace IDs

Setting up tracing adds an identifier, or trace ID, to all of these events.
The trace ID generates when the request initiates.
That same trace ID applies to every span as the request and response generate activity across the system.

The trace ID lets you trace, or follow, a request as it flows from node to node, service to microservice to lambda function to wherever it goes in your chaotic, cloud computing system and back again.
This is recorded and displayed as spans.

Here's an example showing two pages in Grafana Cloud.
The first, numbered 1, shows a query using the **Explore** feature.
In the query results, you can see a **TraceID** field that was added to an application.
That field contains a **Tempo** trace ID.
The second page, numbered 2, uses the same **Explore** feature to perform a Tempo search using that **TraceID**.
It then shows a set of spans as horizontal bars, each bar denoting a different part of the system.

![Traces example with query results and spans](/static/img/docs/tempo/screenshot-trace-explore-spans-g10.png)

## What are traces used for?

Traces can help you find bottlenecks.
Applications like Grafana can visualize traces to give a graphic representation of how long it takes for each step in the data flow pathway to complete.
It can show where additional requests initiate and end, and how your system responds.
This data helps you locate problem areas, often in places you never would have anticipated or found without this ability to trace the request flow.

<!-- What traces provide that logs and metrics don't -->
{{< youtube id="0tlp7QCPu0k" >}}

## Learn more

For more information about traces, refer to:

* [Traces and telemetry](./telemetry)
* [User journeys: How tracing can help you](./solutions-with-traces)
* [Glossary](./glossary)