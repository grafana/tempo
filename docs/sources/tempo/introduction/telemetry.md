---
aliases:
  - /docs/tempo/getting-started/traces
  - /docs/tempo/traces
description: "Traces and telemetry"
keywords:
  - Grafana
  - traces
  - tracing
title: Traces and telemetry
weight: 120
---

# Traces and telemetry

Metrics, logs, traces, and profiles form the pillars of observability.

SCREENSHOT

Metrics provide the health data about the state of a system.
Often, metrics are the first alert that something is going on and are where discovery first starts. Metrics indicate that something is happening.

Logs provide an audit trail of activity from a single process that create an informational context.
Logs act as atomic events, detailing what is occurring in the services in your application.
Log lines can quickly show you the type of errors that are occurring, or give you debug information on situations that are occurring at a point in time.
Logs let you know what is happening to your application.

Traces add further to the observability picture by telling you what happens at each step or action in a data pathway. Traces provide the map–-the where–-something is going wrong.
A trace can be visualized to give a graphic representation of how long it takes for each step in the data flow pathway, such as HTTP requests, to complete.
It can show where new requests are initiated and finished, as well as how your system responds.
This data helps you locate problem areas and assess their impact, often in places you never would have anticipated or found without this ability to trace the request flow.

Profiles narrow down issues in your application codebase, showing you how resources such as CPU time and memory are being utilized and where in your code this occurs.
This allows you to get to specific lines of code that can be optimized.

Correlating between the four pillars of observability helps create a full holistic view of your application and infrastructure.

## Why traces?

Metrics in themselves aren't sufficient to find the root cause and solve complex issues.
The same can be said for logs, which can contain a significant amount of information but lack the context of the interactions and dependencies between the different components of your complex environment.
Each of those pillars of observability (metrics, logs, traces) provide their own value.
To get the most value of your observability strategy, you need to be able to correlate them.

Often products will do a correlative analysis on metrics to attempt to find signals that correlate with our failing edge endpoint such as a dependent database.
Traces have the unique ability to show true causative relationships between services.
You can directly see the failing database and all impacted failing edge endpoints.

Using traces and [exemplars](https://grafana.com/docs/grafana/next/fundamentals/exemplars/), you can go from a metric data point and get to an associated trace.

SCREENSHOT

Or from traces to logs

SCREENSHOT

 ...and vice versa, from logs to traces

 SCREENSHOT

## Use tracing data to understand your services

Tracing is best used for analyzing the performance of your system, identifying bottlenecks, monitoring latency, and providing a complete picture of how requests are processed.

* Decrease MTTR/MTTI: Tracing helps reduce Mean Time To Repair (MTTR) and Mean Time To Identify (MTTI) by pinpointing exactly where errors or latency are occurring within a transaction across multiple services.
* Optimization of bottlenecks and long-running code: By visualizing the path and duration of requests, tracing can help identify bottleneck operations and long-running pieces of code that could benefit from optimization.
* Metrics generation and RED signals: Tracing can help generate useful metrics related to Request rate, Error rate, and  Duration of requests (RED). These high-level signals can quickly highlight problem areas in your system.
* Seamless telemetry correlation Event history correlated to logs and metrics: Using tracing in conjunction with logs and metrics can help give you a comprehensive view of events overtime during incident response and postmortems by showing relationships between services and dependencies.
* Business policy adherence ensures that services are correctly isolated using generated metrics and generated service graphs.


<!-- What traces provide that logs and metrics don't -->
{{< youtube id="0tlp7QCPu0k" >}}

