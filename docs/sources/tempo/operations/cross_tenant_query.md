---
title: Cross-tenant query federation
menuTitle: Cross-tenant query
description: Cross-tenant query federation
weight: 70
aliases:
- /docs/tempo/operations/cross-tenant-query
---


# Cross-tenant query federation

{{% admonition type=note" %}}
You need to enable `multitenancy_enabled: true` in the cluster for multi-tenant querying to work.
see [enable multi-tenancy]({{< relref "./multitenancy" >}}) for more details and implications of `multitenancy_enabled: true`.
{{% /admonition %}}

Tempo supports multi-tenant queries for search, search-tags and trace-by-id search operations.

To perform multi-tenant queries, send tenant IDs separated by a `|` character in the `X-Scope-OrgID` header, for e.g: `foo|bar`.

By default, Cross-tenant query is enabled and can be controlled using `multi_tenant_queries_enabled` configuration setting.

```yaml
query_frontend:
   multi_tenant_queries_enabled: true
```

Queries performed using the cross-tenant configured data source, in either **Explore** or inside of dashboards, 
are performed across all the tenants that you specified in the **X-Scope-OrgID** header. 
