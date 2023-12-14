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

Tempo supports multi-tenant queries. where users can send list of tenants  multiple tenants.

The tenant IDs involved need to be specified separated by a '|' character in the 'X-Scope-OrgID' header.

cross-tenant query is enabled by default, and can be controlled using `multi_tenant_queries_enabled` config.

```yaml
query_frontend:
   multi_tenant_queries_enabled: true
```

### Use cross-tenant query federation

To submit a query across all tenants that your access policy has access rights to, you need to configure tempo datasource.

Update Tempo datasource to send `X-Scope-OrgID` header with values of tenants separated by `|` e.g. `test|test1`, and query the tempo like you already do.

<p align="center"><img src="../header_ds.png" alt="X-Scope-OrgID Headers in Datasource"></p>

If you are provisioning tempo datasource via Grafana Provisioning, you can configure `x-scope-orgid` header like this:

```yaml
  jsonData:
    httpHeaderName1: 'x-scope-orgid'
  secureJsonData:
    httpHeaderValue1: 'test|test1'
```

> NOTE: for streaming search, header `x-scope-orgid` needs to be lowercase.

Queries are performed using the cross-tenant configured data source in either **Explore** or inside of dashboards are performed across all the tenants that you specified in the **X-Scope-OrgID** header. 

These queries are processed as if all the data were in a single tenant.

Tempo will inject `tenant` resource in the responses to show which tenant the trace came from:

<p align="center"><img src="../multi_tenant_trace.png" alt="tenant resource attribute in response trace"></p>

