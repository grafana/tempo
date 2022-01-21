---
title: Tempo in Grafana
weight: 300
---

# Tempo in Grafana

Grafana has a built-in Tempo datasource that can be used to query Tempo and visualize traces.  This page describes the high-level features and their availability.  Use the latest versions for best compatibility and stability.

## View trace by ID
The most basic functionality is to visualize a trace using its ID.  Select the Trace ID tab and enter the ID to view it. This functionality is enabled by default and is available in all versions of Grafana.
<p align="center"><img src="../grafana-query.png" alt="View trace by ID"></p>

## Log search
Traces can be discovered by searching logs for entries containing trace IDs.  This is most useful when your application also logs relevant information about the trace that can also be searched, such as HTTP status code, customer ID, etc.  This feature requires Grafana 7.5 or later, with a linked Loki data source, and a [traceID derived field](https://grafana.com/docs/grafana/latest/datasources/loki/#derived-fields).

<p align="center"><img src="../log-search.png" alt="Log Search"></p>


## Tempo search
<span style="background-color:#f3f973;">Tempo search is an experimental feature.</span>

### Search of recent traces

Tempo includes the ability to search recent traces held in ingesters.
Traces can be searched for data originating from a specific service,
duration range, span, or process-level attributes included in your application's instrumentation, such as HTTP status code and customer ID.
Search of recent traces is disabled by default.
Ingesters default to storing the last 15 minutes of traces.

To enable recent traces search:
-  Run Tempo, enabling search in the YAML configuration.
Refer to the [search](../../configuration#search) configuration documentation.
-  Run Grafana 8.2 or a more recent version. Enable the `tempoSearch` [feature toggle](https://github.com/grafana/tempo/blob/main/example/docker-compose/tempo-search/grafana.ini).

<p align="center"><img src="../tempo-search.png" alt="Tempo Search"></p>

### Search of the backend datastore

Tempo includes the the ability to search the entire backend datastore.

To enable search of the backend datastore:
-  Run Tempo, enabling search in the YAML configuration.
Refer to the [search](../../configuration#search) configuration documentation.
Further configration information is in [backend search](../../operations/backend_search).
The Tempo configuration is the same for searching recent traces or
for search of the backend datastore. 
-  Run Grafana 8.3.4 or a more recent version. Enable the `tempoBackendSearch` [feature toggle](https://github.com/grafana/tempo/blob/main/example/docker-compose/tempo-search/grafana.ini). This will cause Grafana to pass the `start` and `end` parameters necessary for the backend datastore search.

## View JSON file
A local JSON file containing a trace can be uploaded and viewed in the Grafana UI.  This is useful in cases where access to the original Tempo datasource is limited, or for preserving traces outside of Tempo.  The JSON data can be downloaded via the Tempo API or the Inspector panel while viewing the trace in Grafana.

