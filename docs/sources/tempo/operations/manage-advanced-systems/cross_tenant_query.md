---
title: Cross-tenant query federation
menuTitle: Cross-tenant query
description: Cross-tenant query federation
weight: 300
aliases:
  - ../cross-tenant-query # https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/cross-tenant-query/
  - ../cross_tenant_query # https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/cross_tenant_query/
---

# Cross-tenant query federation

Cross-tenant query federation lets you run a single query across multiple tenants at the same time, instead of repeating the same query for each tenant individually.
This is useful when you operate many teams or environments and need to troubleshoot, compare behavior, or enforce governance consistently across them.

## Supported operations

Tempo supports multi-tenant queries for search, search-tags, and trace-by-ID search operations.
TraceQL metrics queries (for example, query range and instant endpoints that use TraceQLMetrics) also support cross-tenant queries.

These operations can be federated across multiple tenants when you specify more than one tenant ID in the `X-Scope-OrgID` header.

To perform multi-tenant queries, send tenant IDs separated by a `|` character in the `X-Scope-OrgID` header, for example, `foo|bar`.

## Enable cross-tenant query

By default, cross-tenant query is enabled and can be controlled using `multi_tenant_queries_enabled` configuration setting.

This setting is a no-op unless `multitenancy_enabled: true` is enabled in the cluster configuration.

```yaml
query_frontend:
  multi_tenant_queries_enabled: true
```

For more information on configuration options, refer to [Enable multi-tenancy](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/manage-advanced-systems/multitenancy/).

## TraceQL queries

Queries performed using the cross-tenant configured data source, in either **Explore** or inside of dashboards,
are performed across all the tenants that you specified in the **X-Scope-OrgID** header.

TraceQL queries that compare multiple spansets may not correctly return all traces in a cross-tenant query. For instance,

```
{ span.attr1 = "bar" } && { span.attr2 = "foo" }
```

TraceQL evaluates a contiguously stored trace.
If these two conditions are satisfied in separate tenants, then Tempo doesn't correctly return the trace.

## Tenant IDs in cross-tenant queries

Tempo uses the same `X-Scope-OrgID` header for cross-tenant queries as for single-tenant queries.
To query multiple tenants, specify tenant IDs separated by a `|` character in the header.

When multiple tenant IDs are specified, Tempo normalizes them by sorting and removing duplicate tenant IDs.
For example, `bar|foo|bar` is normalized to `bar|foo`.

The following example shows an HTTP request that performs a cross-tenant query across the `foo` and `bar` tenants:

```bash
curl -H "X-Scope-OrgID: foo|bar" \
  "http://tempo-query:3200/api/traces/search?limit=20&query={ service.name = \"checkout\" }"
```

For tenant ID format, allowed characters, and reserved values, refer to [Tenant IDs](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/tenant-ids/).

### Security considerations

The `X-Scope-OrgID` header should be set by a trusted reverse proxy, not by client applications. Tempo assumes that clients setting `X-Scope-OrgID` are trusted, and the responsibility of populating this value should be handled by your authenticating reverse proxy.

Allowing clients to set the tenant ID directly can lead to unauthorized access to other tenants' data. A malicious client could change the tenant ID in the header to access traces from other tenants.

For more information about setting up authentication and reverse proxies, refer to [Manage authentication](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/authentication/).
