---
title: Cross-tenant query federation
menuTitle: Cross-tenant query
description: Cross-tenant query federation
weight: 70
aliases:
- /docs/tempo/operations/cross-tenant-query
---


# Cross-tenant query federation

{{% admonition type="note" %}}
You need to enable `multitenancy_enabled: true` in the cluster for multi-tenant querying to work.
Refer to [Enable multi-tenancy](/docs/tempo/latest/operations/multitenancy/) for more details and implications of `multitenancy_enabled: true`.
{{% /admonition %}}

Tempo supports multi-tenant queries for search, search-tags, and trace-by-ID search operations.

To perform multi-tenant queries, send tenant IDs separated by a `|` character in the `X-Scope-OrgID` header, for example, `foo|bar`.

By default, cross-tenant query is enabled and can be controlled using `multi_tenant_queries_enabled` configuration setting.

```yaml
query_frontend:
   multi_tenant_queries_enabled: true
```

For more information on configuration options, refer to [Enable multitenancy](https://grafana.com/docs/tempo/latest/operations/multitenancy/).

## TraceQL queries

Queries performed using the cross-tenant configured data source, in either **Explore** or inside of dashboards,
are performed across all the tenants that you specified in the **X-Scope-OrgID** header.

TraceQL queries that compare multiple spansets may not correctly return all traces in a cross-tenant query. For instance,

```
{ span.attr1 = "bar" } && { span.attr2 = "foo" }
```

TraceQL evaluates a contiguously stored trace.
If these two conditions are satisfied in separate tenants, then Tempo  doesn't correctly return the trace.
