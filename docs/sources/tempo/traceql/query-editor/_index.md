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

You can compose TraceQL queries in Grafana and Grafana Cloud using **Explore** and a Tempo data source. You can use either the **Query type** > **Search** (the TraceQL query builder) or the **TraceQL** tab (the TraceQL query editor).
Both of these methods let you build queries and drill-down into result sets.

To add TraceQL panels to your dashboard, refer to the [Traces panel documentation](/docs/grafana/latest/panels-visualizations/visualizations/traces/).

To learn more about Grafana dashboards, refer to the [Use dashboards documentation](/docs/grafana/latest/dashboards/use-dashboards/).

## TraceQL query builder

The TraceQL query builder, located on the **Explore** > **Query type** > **Search** in Grafana, provides drop-downs and text fields to help you write a query.

Refer to the [Search using the TraceQL query builder documentation]({{< relref "./traceql-search" >}}) to learn more about creating queries using convenient drop-down menus.

![The TraceQL query builder](/static/img/docs/tempo/screenshot-traceql-query-type-search-v10.png)


## TraceQL query editor

The TraceQL query editor, located on the **Explore** > **TraceQL** tab in Grafana, lets you search by trace ID and write TraceQL queries using autocomplete.

Refer to the [TraceQL query editor documentation]({{< relref "./traceql-editor" >}}) to learn more about constructing queries using a code-editor-like experience.

![The TraceQL query editor](/static/img/docs/tempo/screenshot-traceql-query-editor-v10.png)
