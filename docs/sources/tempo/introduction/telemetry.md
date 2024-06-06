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
weight: 200
---

# Traces and telemetry

Metrics, logs, traces, and profiles form the pillars of observability.
Correlating between the four pillars of observability helps create a full holistic view of your application and infrastructure.

![The four pillars of observability](/media/docs/tempo/intro/four-pillars-observe.png.png)

## Metrics

Metrics provide the health data about the state of a system.
Often, metrics are the first alert that something is going on and are where discovery first starts.
Metrics indicate that something is happening.

## Logs

Logs provide an audit trail of activity from a single process that create an informational context.
Logs act as atomic events, detailing what's occurring in the services in your application.
Log lines can quickly show you the type of errors that are occurring, or give you debug information on situations that are occurring at a point in time.
Logs let you know what's happening to your application.


## Traces

Traces add further to the observability picture by telling you what happens at each step or action in a data pathway. Traces provide the map–-the where–-something is going wrong.
A trace can be visualized to give a graphic representation of how long it takes for each step in the data flow pathway, such as HTTP requests, to complete.
It can show where new requests are initiated and finished, as well as how your system responds.
This data helps you locate problem areas and assess their impact, often in places you never would have anticipated or found without this ability to trace the request flow.

## Profiles

Profiles narrow down issues in your application codebase, showing you how resources such as CPU time and memory are being utilized and where in your code this occurs.
This allows you to get to specific lines of code that can be optimized.

## Why traces?

Metrics in themselves aren't sufficient to find the root cause and solve complex issues.
The same can be said for logs, which can contain a significant amount of information but lack the context of the interactions and dependencies between the different components of your complex environment.
Each of those pillars of observability (metrics, logs, traces, profiles) provide their own value.
To get the most value of your observability strategy, you need to be able to correlate them.

Often products will do a correlative analysis on metrics to attempt to find signals that correlate with our failing edge endpoint such as a dependent database.
Traces have the unique ability to show true causative relationships between services.
You can directly see the failing database and all impacted failing edge endpoints.

Using traces and [exemplars](https://grafana.com/docs/grafana/next/fundamentals/exemplars/), you can go from a metric data point and get to an associated trace.

![Use exemplars to go from a metric data point to a trace](/media/docs/tempo/intro/exemplar-metric-totrace.png)

Or from traces to logs:

![Use traces to go to a log entry](/media/docs/tempo/intro/tempo-logs-to-traces.png)

And vice versa, from logs to traces.

![Use logs to go to a span](/media/docs/tempo/intro/loki-trace-to-logspng.png)
