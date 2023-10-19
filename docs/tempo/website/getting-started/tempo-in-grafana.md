---
aliases:
- /docs/tempo/v1.2.1/getting-started/tempo-in-grafana/
title: Tempo in Grafana
weight: 300
---

# Tempo in Grafana

[Grafana 7.4](https://grafana.com/grafana/download/7.4.5) and later have a built in Tempo datasource that can be used to query Tempo and visualize traces.  This page describes the high-level features and their availability.  Use the latest versions for best compatibility and stability.

1. [View trace by ID](#view-trace-by-id)
2. [Log Search](#log-search)
3. [Tempo Search](#tempo-search)
4. [View JSON file](#view-json-file)

## View trace by ID
The most basic functionality is to visualize a trace using its ID.  Select the Trace ID tab and enter the ID to view it. This functionality is enabled by default and is available in all versions of Grafana.
<p align="center"><img src="../grafana-query.png" alt="View trace by ID"></p>

## Log search
Traces can be discovered by searching logs for entries containing trace IDs.  This is most useful when your application also logs relevant information about the trace that can also be searched, such as HTTP status code, customer ID, etc.  This feature requires Grafana 7.5 or later, with a linked Loki data source, and a [traceID derived field](https://grafana.com/docs/grafana/latest/datasources/loki/#derived-fields).

<p align="center"><img src="../log-search.png" alt="Log Search"></p>


## Tempo search
<span style="background-color:#f3f973;">This experimental feature is disabled by default. See below for more information on how to enable.</span>

Tempo includes native search of recent traces.  Traces can be searched for data originating from a specific service, duration range, and span and process-level attributes included in your application's instrumentation, such as HTTP status code, customer ID, etc.  Currently only search of traces at the ingesters is supported. By default the ingesters store the last 15 minutes.

Enabling this feature requires the following:
1. Run the Tempo 1.2 or the latest pre-release and enable search in the YAML config. For more information see the [Tempo configuration documentation](../../configuration#search).
2. Run Grafana 8.2 or the latest pre-release and enable the `tempoSearch` [feature toggle](https://github.com/grafana/tempo/blob/main/example/docker-compose/tempo-search/grafana.ini).

<p align="center"><img src="../tempo-search.png" alt="Tempo Search"></p>


## View JSON file
A local JSON file containing a trace can be uploaded and viewed in the Grafana UI.  This is useful in cases where access to the original Tempo datasource is limited, or for preserving traces outside of Tempo.  The JSON data can be downloaded via the Tempo API or the Inspector panel while viewing the trace in Grafana.

