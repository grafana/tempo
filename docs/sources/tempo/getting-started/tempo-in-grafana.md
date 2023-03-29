---
title: Tempo in Grafana
weight: 400
---

# Tempo in Grafana

Grafana has a built-in Tempo datasource that can be used to query Tempo and visualize traces.  This page describes the high-level features and their availability.  Use the latest versions for best compatibility and stability.

## View trace by ID

The most basic functionality is to visualize a trace using its ID.  Select the Trace ID tab and enter the ID to view it. This functionality is enabled by default and is available in all versions of Grafana.
<p align="center"><img src="../assets/grafana-query.png" alt="View trace by ID"></p>

## Log search

Traces can be discovered by searching logs for entries containing trace IDs.  This is most useful when your application also logs relevant information about the trace that can also be searched, such as HTTP status code, customer ID, etc.  This feature requires Grafana 7.5 or later, with a linked Loki data source, and a [traceID derived field](https://grafana.com/docs/grafana/latest/datasources/loki/#derived-fields).

<p align="center"><img src="../assets/log-search.png" alt="Log Search"></p>


## Use TraceQL to dig deep into trace data

Inspired by PromQL and LogQL, TraceQL is a query language designed for selecting traces in Tempo.

The default Tempo search reviews the whole trace. TraceQL provides a method for formulating precise queries so you can quickly identify the traces and spans that you need. Query results are returned faster because the queries limit what is searched.

You can run a TraceQL query either by issuing it to Tempo’s `q` parameter of the [`search` API endpoint]({{< relref "../api_docs/#search" >}}), or, for those using Tempo in conjunction with Grafana, by using Grafana’s [TraceQL query editor]({{< relref "../traceql/query-editor" >}}).

For details about how queries are constructed, read the [TraceQL documentation]({{< relref "../traceql" >}}).

## Find traces using Tempo tags search

Search for traces using common dimensions such as time range, duration, span tags, service names, and more. Use the trace view to quickly diagnose errors and high-latency events in your system.

### Non-deterministic search

Most search functions are deterministic: using the same search criteria results in the same results.

However, Tempo search is non-deterministic.
If you perform the same search twice, you’ll get different lists, assuming the possible number of results for your search is greater than the number of results you have your search set to return.

When performing a search, Tempo does a massively parallel search over the given time range, and takes the first N results. Even identical searches will differ due to things like machine load and network latency. This approach values speed over predictability and is quite simple; enforcing that the search results are consistent would introduce additional complexity (and increase the time the user spends waiting for results). TraceQL follows the same behavior.

## Service graph view

Grafana provides a built-in service graph view available in Grafana Cloud and Grafana 9.1.
The service graph view visualizes the span metrics (traces data for rates, error rates, and durations (RED)) and service graphs.
Once the requirements are set up, this pre-configured view is immediately available in **Explore > Service Graphs**.

For more information, refer to the [service graph view]({{< relref "../metrics-generator/service-graph-view/" >}}).

<p align="center"><img src="../assets/apm-overview.png" alt="Service graph view overview"></p>

## Metrics from spans

RED metrics can be used to drive service graphs and other ready-to-go visualizations of your span data. RED metrics represent:

- Rate, the number of requests per second
- Errors, the number of those requests that are failing
- Duration, the amount of time those requests take

For more information about RED method, refer to [The RED Method: How to instrument your services](https://grafana.com/blog/2018/08/02/the-red-method-how-to-instrument-your-services/).

>**Note:** Metrics generation is disabled by default. Contact Grafana Support to enable metrics generation in your organization.

After the metrics generator is enabled in your organization, refer to [Metrics-generator configuration]({{< relref "../configuration">}}) for information about metrics-generator options.

<p align="center"><img src="../assets/trace_service_graph.png" alt="Trace service graph"></p>

These metrics exist in your Hosted Metrics instance and can also be easily used to generate powerful custom dashboards.

<p align="center"><img src="../assets/trace_custom_metrics_dash.png" alt="Trace custom metrics dashboard"></p>

The metrics generator automatically generates exemplars as well which allows easy metrics to trace linking. [Exemplars](https://grafana.com/docs/grafana-cloud/data-configuration/traces/exemplars/) are GA in Grafana Cloud so you can also push your own.

<p align="center"><img src="../assets/trace_exemplars.png" alt="Trace exemplars"></p>

## View JSON file
A local JSON file containing a trace can be uploaded and viewed in the Grafana UI. This is useful in cases where access to the original Tempo data source is limited, or for preserving traces outside of Tempo. The JSON data can be downloaded via the Tempo API or the Inspector panel while viewing the trace in Grafana.

## Linking traces and metrics

Grafana can correlate different signals by adding the functionality to link between traces and metrics. The [trace to metrics feature](https://grafana.com/blog/2022/08/18/new-in-grafana-9.1-trace-to-metrics-allows-users-to-navigate-from-a-trace-span-to-a-selected-data-source/), a beta feature in Grafana 9.1, lets you quickly see trends or aggregated data related to each span.

You can try it out by enabling the `traceToMetrics` feature toggle in your Grafana configuration file.

For example, you can use span attributes to metric labels by using the `$__tags` keyword to convert span attributes to metrics labels.

For more information, refer to the [trace to metric configuration](https://grafana.com/docs/grafana/latest/datasources/tempo/#trace-to-metrics) documentation.
