---
title: Write TraceQL queries in Grafana
menuTitle: Write TraceQL queries in Grafana
description: Learn how to create TraceQL queries in Grafana using the query editor and search.
aliases:
  - /docs/tempo/latest/traceql/construct-query
  - /docs/tempo/latest/traceql/query-editor/traceql-editor
  - /docs/tempo/latest/traceql/query-editor/traceql-search
weight: 400
keywords:
  - Tempo query language
  - query editor
  - TraceQL
---

# Write TraceQL queries in Grafana

The Tempo data source's query editor helps you query and display traces from Tempo in [Explore](https://grafana.com/docs/grafana/<GRAFANA_VERSION>/explore/).
The queries use [TraceQL](/docs/tempo/latest/traceql), the query language designed specifically for tracing.

For general documentation on querying data sources in Grafana, refer to [Query and transform data](/docs/grafana/<GRAFANA_VERSION>/panels-visualizations/query-transform-data/).

## Before you begin

You can compose TraceQL queries in Grafana and Grafana Cloud using **Explore** and a Tempo data source.

## Choose a query editing mode

The query editor has three modes, or **Query types**, that you can use to explore your tracing data.
You can use these modes by themselves or in combination to create building blocks to generate custom queries.

![The three query types: Search, TraceQL, and Service Graph](/media/docs/grafana/data-sources/tempo/query-editor/tempo-ds-query-types.png)

The three **Query types** are:

- The **Search** query builder provides a user interface for building a TraceQL query.
- The **TraceQL** query editor lets you write your own TraceQL query with assistance from autocomplete.
- The **Service Graph** view displays a visual relationship between services. Refer to the [Service graph view](https://grafana.com/docs/tempo/<TEMPO_VERSION>/metrics-generator/service-graph-view/) documentation for more information.

### Search query builder

The Search query builder provides drop-down lists and text fields to help you write a query.
The query builder is ideal for people who aren't familiar with or want to learn TraceQL.

Refer to the [Search using the TraceQL query builder documentation](https://grafana.com/docs/grafana/<GRAFANA_VERSION>/datasources/tempo/query-editor/traceql-search/) to learn more about creating queries using convenient drop-down menus.

![The Search query builder](/media/docs/grafana/data-sources/tempo/query-editor/tempo-ds-query-search-v11.png)

### TraceQL query editor

The TraceQL query editor lets you search by trace ID and write TraceQL queries using autocomplete.

Refer to the [TraceQL query editor documentation](https://grafana.com/docs/grafana/<GRAFANA_VERSION>/datasources/tempo/query-editor/traceql-editor/) to learn more about constructing queries using a code-editor-like experience.

![The TraceQL query editor](/media/docs/grafana/data-sources/tempo/query-editor/tempo-ds-query-traceql-v11.png)

You can also search for a Trace ID by entering a trace ID into the query field.

### Service graph view

Grafana’s service graph view uses metrics to display span request rates, error rates, and durations, as well as service graphs.
Once the requirements are set up, this pre-configured view is immediately available.

Using the service graph view, you can:

- Discover spans which are consistently erroring and the rates at which they occur
- Get an overview of the overall rate of span calls throughout your services
- Determine how long the slowest queries in your service take to complete
- Examine all traces that contain spans of particular interest based on rate, error, and duration values (RED signals)

For more information about the service graph, refer to [Service graph view](https://grafana.com/docs/tempo/<TEMPO_VERSION>/metrics-generator/service-graph-view/).

![Screenshot of the Service Graph view](/media/docs/grafana/data-sources/tempo/query-editor/tempo-ds-query-service-graph.png)

## Use TraceQL panels in dashboards

To add TraceQL panels to your dashboard, refer to the [Traces panel documentation](/docs/grafana/latest/panels-visualizations/visualizations/traces/).

To learn more about Grafana dashboards, refer to the [Use dashboards documentation](/docs/grafana/latest/dashboards/use-dashboards/).