---
title: TraceQL query editor
menuTitle: TraceQL query editor
description: Learn how to use the TraceQL query editor
aliases:
  - /docs/tempo/latest/traceql/construct-query
weight: 400
keywords:
  - Tempo query language
  - query editor
  - TraceQL
---

# TraceQL query editor

With Tempo 2.0, you can use the TraceQL query editor in the Tempo data source to build queries and drill-down into result sets. The editor is available in Grafana’s Explore interface.


<p align="center"><img src="../assets/query-editor-http-method.png" alt="Query editor showing request for http.method" /></p>

Using the query editor, you can use the editor’s autocomplete suggestions to write queries. The editor detects span sets to provide relevant autocomplete options. It uses regular expressions (regex) to detect where it is inside a spanset and provide attribute names, scopes, intrinsic names, logic operators, or attribute values from Tempo's API, depending on what is expected for the current situation.

<p align="center"><img src="../assets/query-editor-auto-complete.png" alt="Query editor showing the auto-complete feature" /></p>

Query results are returned in a table. Selecting the Trace ID or Span ID provides more detailed information.

<p align="center"><img src="../assets/query-editor-results-span.png" alt="Query editor showing span results" /></p>

Selecting the trace ID from the returned results will open a trace diagram. Selecting a span from the returned results opens a trace diagram and reveals the relevant span in the trace diagram (above, the highlighted blue line).
