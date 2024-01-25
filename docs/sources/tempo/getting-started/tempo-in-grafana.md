---
title: Tempo in Grafana
description: Grafana has a built-in Tempo data source that can be used to query Tempo and visualize traces.
weight: 400
---

# Tempo in Grafana

Grafana has a built-in Tempo data source that can be used to query Tempo and visualize traces.
This page describes the high-level features and their availability.
Use the latest versions for best compatibility and stability.

## Use TraceQL to dig deep into trace data

Inspired by PromQL and LogQL, TraceQL is a query language designed for selecting traces in Tempo.

The default Tempo search reviews the whole trace. TraceQL provides a method for formulating precise queries so you can quickly identify the traces and spans that you need. Query results are returned faster because the queries limit what is searched.

You can run a TraceQL query either by issuing it to Tempo’s `q` parameter of the [`search` API endpoint]({{< relref "../api_docs#search" >}}), or, for those using Tempo in conjunction with Grafana, by using Grafana’s [TraceQL query editor]({{< relref "../traceql/query-editor" >}}).

For details about how queries are constructed, read the [TraceQL documentation]({{< relref "../traceql" >}}).

<p align="center"><img src="../../traceql/assets/query-editor-results-span.png" alt="Query editor showing span results" /></p>

The most basic functionality is to visualize a trace using its ID. Select the TraceQL tab and enter the ID to view it. This functionality is enabled by default and is available in all versions of Grafana.

## Finding traces using Loki logs

Traces can be discovered by searching logs for entries containing trace IDs.  This is most useful when your application also logs relevant information about the trace that can also be searched, such as HTTP status code, customer ID, etc.  This feature requires Grafana 7.5 or later, with a linked Loki data source, and a [traceID derived field](/docs/grafana/latest/datasources/loki/#derived-fields).

## Find traces using Tempo tags search

Search for traces using common dimensions such as time range, duration, span tags, service names, and more. Use the trace view to quickly diagnose errors and high-latency events in your system.

<p align="center"><img src="../../traceql/assets/screenshot-explore-traceql-search.png" alt="Showing how to build queries with common dimensions using query builder" /></p>

### Non-deterministic search

Most search functions are deterministic: using the same search criteria results in the same results.

However, Tempo search is non-deterministic.
If you perform the same search twice, you’ll get different lists, assuming the possible number of results for your search is greater than the number of results you have your search set to return.

When performing a search, Tempo does a massively parallel search over the given time range, and takes the first N results. Even identical searches will differ due to things like machine load and network latency. This approach values speed over predictability and is quite simple; enforcing that the search results are consistent would introduce additional complexity (and increase the time the user spends waiting for results). TraceQL follows the same behavior.

## Service graph view

Grafana provides a built-in service graph view available in Grafana Cloud and Grafana 9.1.
The service graph view visualizes the span metrics (traces data for rates, error rates, and durations (RED)) and service graphs.
Once the requirements are set up, this pre-configured view is immediately available in **Explore > Service Graphs**.

For more information, refer to the [service graph view]({{< relref "../metrics-generator/service-graph-view" >}}).

<p align="center"><img src="../assets/apm-overview.png" alt="Service graph view overview"></p>

## View JSON file

A local JSON file containing a trace can be imported and viewed in the Grafana UI. This is useful in cases where access to the original Tempo data source is limited, or for preserving traces outside of Tempo.

The JSON data can be downloaded via the Tempo API or the [Inspector panel](/docs/grafana/latest/explore/explore-inspector/) while viewing the trace in Grafana.

{{% admonition type="note" %}}
To perform this action on Grafana 10.1 or later, select a Tempo data source, select **Explore** from the main menu, and then select **Import trace**.
{{% /admonition %}}

## Link tracing data with profiles

Using Trace to profiles, you can use Grafana’s ability to correlate different signals by adding the functionality to link between traces and profiles.

Trace to profiles lets you link your [Grafana Pyroscope](/docs/pyroscope/) data source to tracing data in Grafana or Grafana Cloud.
When configured, this connection lets you run queries from a trace span into the profile data.

For more information, refer to the [Traces to profiles documentation](docs/grafana/latest/datasources/tempo/configure-tempo-data-source/#trace-to-profiles) and the [Grafana Pyroscope data source documentation](/docs/grafana/latest/datasources/grafana-pyroscope/).

{{< youtube id="AG8VzfFMLxo" >}}