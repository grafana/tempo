---
title: Tune TraceQL query performance
menuTitle: Tune query performance
description: Practical ways to make TraceQL queries faster in Tempo
weight: 310
keywords:
  - TraceQL
  - performance
  - optimization
---

# Tune TraceQL query performance

Use these techniques to make TraceQL queries faster and more cost‑efficient in Tempo. The guidance assumes you already understand tracing concepts and how to write basic TraceQL queries.

Before you begin, ensure you have the following:

- A running Tempo deployment and access to query it from Grafana
- Familiarity with TraceQL expressions and pipelines
- Recent Tempo storage formats (for example, vParquet4+) for best performance

## Use only and (&&) when possible

Queries that consist only of logical and (`&&`) conditions are much faster than queries that include other logical or structural operators (`||`, `>>`, and so on). These queries let the TraceQL engine push filtering down into the Parquet layer (`predicate pushdown`), which is the most efficient path.

Rewrite or simplify queries to use `&&` when possible.

For example, to match spans with a non‑200 status code on a specific path, use a single conjunction inside one selector:

The following query returns a single span where the HTTP status code is not `200` and the URL is `/api`:

```traceql
{ span.http.status_code != 200 && span.http.url = "/api" }
```

Also avoid splitting these conditions across multiple selectors joined by `&&`, which matches them on different spans and changes semantics:

```traceql
{ span.http.url = "/api" } && { span.http.status_code != 200 }
```

Use a single selector when you want both conditions to be true on the same span.

```traceql
{ span.http.url = "/api" && span.http.status_code != 200 }
```

## Prefer scoped attributes over attributes without a scope

Always scope attributes with `span.`, `resource.`, `event.`, or `link.`. Attributes without a scope make Tempo check in multiple places (for example, span and resource), which is slower.

To find spans by HTTP status code, prefer a scoped attribute:

The following query scopes the attribute to the span and runs faster than an equivalent without a scope:

```traceql
{ span.http.status_code = 500 }
```

Avoid forms without a scope that force extra lookups:

```traceql
{ .http.status_code = 500 }
```

## Access as few attributes as possible

Tempo stores trace data in a columnar format (Parquet). It only reads the columns referenced by your query. If you can achieve the same effect while referencing fewer attributes, the query runs faster.

For example, if filtering by status code already identifies error spans in your environment, you can drop a redundant status check.

This query filters on both an HTTP status and an explicit status:

```traceql
{ span.http.status_code = 500 && status = error }
```

If the status code alone is sufficient, prefer the simpler form:

```traceql
{ span.http.status_code = 500 }
```

## Filter by trace‑ and resource‑level attributes

Columns for trace‑level intrinsic fields such as `trace:duration`, `trace:rootName`, and `trace:rootService`, and for resource attributes such as `resource.service.name`, are much smaller than span‑level columns. Filtering on these fields reduces data that must be scanned.

To find long traces for a given service, filter on a trace intrinsic and a resource attribute:

```traceql
{ trace:duration > 5s && resource.service.name = "api" }
```

To target a specific root operation in production, filter on `trace:rootName` and a resource attribute:

```traceql
{ trace:rootName = "POST /api/orders" && resource.deployment.environment = "production" }
```

## Use dedicated columns for common fields

Tempo exposes many frequently used fields as dedicated columns in Parquet to accelerate filtering and selection. Prefer these well‑known, scoped fields instead of relying on nested or ambiguous attribute paths.

For example, the following queries benefit from dedicated columns and scope:

```traceql
{ span.http.method = "GET" }
{ span.db.system = "postgresql" }
{ resource.cloud.region =~ "us-east-1|us-west-1" }
```

## Use sampling for large metrics queries

For TraceQL metrics queries on very large datasets, enable sampling to return approximate results faster. Sampling can dramatically reduce scan time while retaining useful accuracy for operational dashboards.

To get the 90th percentile span duration for a service with sampling enabled:

```traceql
{ resource.service.name = "api" } | quantile_over_time(duration, 0.9) with(sample=true)
```

For guidance on when and how to use sampling, refer to the [Sampling guide](https://grafana.com/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/instrument-send/set-up-collector/tail-sampling/).

## Next steps

- Build stronger selectors with the [Construct a TraceQL query](https://grafana.com/docs/tempo/<TEMPO_VERSION>/traceql/construct-traceql-queries/) guide.
- Explore aggregations and grouping in [TraceQL metrics queries](https://grafana.com/docs/tempo/<TEMPO_VERSION>/traceql/metrics-queries/).
- Learn about intrinsic fields and attribute scopes in [TraceQL selection and fields](https://grafana.com/docs/tempo/<TEMPO_VERSION>/traceql/construct-traceql-queries/#select-spans).
