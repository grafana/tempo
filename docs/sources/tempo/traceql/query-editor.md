---
title: Write TraceQL queries in Grafana
menuTitle: Write TraceQL queries in Grafana
description: Learn how to create TraceQL queries in Grafana using the query editor and search.
aliases:
  - /docs/tempo/latest/traceql/construct-query
weight: 400
keywords:
  - Tempo query language
  - query editor
  - TraceQL
---

# Write TraceQL queries in Grafana

You can compose TraceQL queries in Grafana and Grafana Cloud using **Explore** and a Tempo data source. You can use either the **Query type** > **Search** or the **TraceQL** tab.
Both of these methods let you build queries and drill-down into result sets.

To add TraceQL panels to your dashboard, refer to the [Traces panel documentation](/docs/grafana/latest/panels-visualizations/visualizations/traces/).

To learn more about Grafana dashboards, refer to the [Use dashboards documentation](/docs/grafana/latest/dashboards/use-dashboards/).

{{% admonition type="note" %}}
To use the TraceQL query editor in Grafana 9.3.2 and newer, you need to enable the `traceqlEditor` feature toggle ([refer to instructions for enabling feature toggles](/docs/grafana/latest/setup-grafana/configure-grafana/feature-toggles/).)

To enable the Trace Search on self managed instances of Grafana 10 and newer, you need to enable the `traceqlSearch` feature flag.

These features are available in Grafana Cloud without enabling a feature flag.
{{% /admonition %}}

## Write TraceQL queries using the query editor

The Tempo data source’s TraceQL query editor helps you query and display traces from Tempo in **Explore**.

To access the query editor, follow these steps:

1. Sign into Grafana or Grafana Cloud.
1. Select your Tempo data source.
1. From the menu, choose **Explore** and select the **TraceQL** tab.
1. Start your query on the text line by entering `{`. For help with TraceQL syntax, refer to the [Construct a TraceQL query documentation]({{< relref "./_index.md" >}}).
1. Optional: Use the Time picker drop-down to change the time and range for the query (refer to the [documentation for instructions](/docs/grafana/latest/dashboards/use-dashboards#set-dashboard-time-range)).
1. Once you have finished your query, select **Run query**.

![Query editor showing request for http.method](/static/img/docs/tempo/query-editor-http-method.png)

### Query by TraceID

To query a particular trace by its trace ID:

1. From the menu, choose **Explore**, select the desired Tempo data source, and navigate to the **TraceQL** tab.
1. Enter the trace ID into the query field. For example: `1f187d8363b5a9b30cedd8e0ce9ccb43`
1. Click **Run query** or use the keyboard shortcut Shift + Enter.

### Use autocomplete to write queries

You can use the query editor’s autocomplete suggestions to write queries.
The editor detects span sets to provide relevant autocomplete options.
It uses regular expressions (regex) to detect where it is inside a spanset and provide attribute names, scopes, intrinsic names, logic operators, or attribute values from Tempo's API, depending on what is expected for the current situation.

To create a query using autocomplete, follow these steps:

1. Use the steps above to access the query editor and begin your query.

1. Enter your query. As you type your query, autocomplete suggestions appear as a drop-down. Each letter you enter refines the autocomplete options to match.

1. Use your mouse or arrow keys to select any option you wish. When the desired option is highlighted, press Tab on your keyboard to add the selection to your query.

1. Once your query is complete, select **Run query** to perform the query.

![Query editor showing the auto-complete feature](/static/img/docs/tempo/query-editor-auto-complete.png)

### View query results

Query results are returned in a table. Selecting the Trace ID or Span ID provides more detailed information.

![Query editor showing span results](/static/img/docs/tempo/query-editor-results-span.png)

Selecting the trace ID from the returned results will open a trace diagram. Selecting a span from the returned results opens a trace diagram and reveals the relevant span in the trace diagram (above, the highlighted blue line).

{{< docs/shared source="grafana" lookup="datasources/tempo-search-traceql.md" version="latest" leveloffset="+1" >}}