---
aliases:
  - /docs/tempo/getting-started/traces
  - /docs/tempo/traces
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
This makes them the ideal observability signal for discovering bottlenecks and interconnection issues.

Traces are composed of one or more spans.
A span is a unit of work within a trace that has a start time relative to the beginning of the trace, a duration and an operation name for the unit of work.
It usually has a reference to a parent span (unless it's the first span, the root span, in a trace).
It frequently includes key/value attributes that are relevant to the span itself, for example the HTTP method used by the span, as well as other metadata such as the service name, sub-span events, or links to other spans.

By definition, traces are never complete. You can always push a new batch of spans, even if days have passed since the last one.
When receiving a query requesting a stored trace, tracing backends like Tempo find all the spans for that specific trace and collate them into a returned result.
For that reason, issues can arise on retrieval of the trace data if traces are extremely large.

<!-- Explanation of traces -->
{{< youtube id="ZirbR0ZJIOs" >}}

## Example of traces

Firstly, a user on your website enters their email address into a form to sign up for your mailing list. They click **Enter**. This initial transaction has a trace ID that's subsequently associated with every interaction in the chain of processes within the system.

Next, the user's email address is data that flows through your system.
In a cloud computing world, it's possible that clicking that one button triggers many downstream processes on various microservices operating across many different nodes in your compute infrastructure. 

As a result, the email address might be sent to a microservice responsible for verification. If the email passes this check, it is then stored in a database.

Along the way, an anonymization microservice strips personally identifying data from the address and adds additional metadata before sending it along to a marketing qualifying microservice which determines whether the request was sent from a targeted part of the internet.

Services respond and data flows back from each, sometimes triggering new events across the system. Along the way, logs are written to the nodes on which those services run with a time stamp showing when the info passed through.

Finally, the request and response activity ends. No other spans are added to that TraceID.

## Traces and trace IDs

Setting up tracing adds an identifier, or trace ID, to all of these events.
The trace ID is generated when the request is initiated and that same trace ID is applied to every single span as the request and response generate activity across the system.

That trace ID enables one to trace, or follow, a request as it flows from node to node, service to microservice to lambda function to wherever it goes in your chaotic, cloud computing system and back again.
This is recorded and displayed as spans.

Here's an example showing two pages in Grafana Cloud. The first, on the left (1), shows a query using the **Explore** feature.
In the query results you can see a **traceID** field that was added to an application. That field contains a **Tempo** trace ID.
The second page, on the right (2), uses the same Explore feature to perform a Tempo search using that **trace ID**.
It then shows a set of spans as horizontal bars, each bar denoting a different part of the system.

![Traces example with query results and spans](/static/img/docs/tempo/screenshot-trace-explore-spans-g10.png)

## What are traces used for?

Traces can help you find bottlenecks.
A trace can be visualized to give a graphic representation of how long it takes for each step in the data flow pathway to complete.
It can show where new requests are initiated and end, and how your system responds.
This data helps you locate problem areas, often in places you never would have anticipated or found without this ability to trace the request flow.

<!-- What traces provide that logs and metrics don't -->
{{< youtube id="0tlp7QCPu0k" >}}

## Learn more

For more information about traces, refer to:

* [Traces and telemetry]({{< relref "./telemetry" >}})
* [User journeys: How tracing can help you]({{< relref "./solutions-with-traces" >}})
* [Glossary]({{< relref "./glossary" >}})