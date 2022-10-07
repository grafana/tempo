---
title: Cross-tenant querying
weight: 600
---

# Cross-tenant querying

Tempo supports creating access policies that can span multiple tenants.
Viewers in Grafana Enterprise can see data coming from more than one tenant simultaneously.

This page covers the ability to query data from multiple tenants at once.

## Prerequisites

- A configured cluster. To create a Tempo cluster,
  refer to [Set up Tempo]({{< relref "../setup" >}}).

- This guide assumes there are two tenants: `team-engineering` and
  `team-finance`. To create a tenant, refer to [Set up a GET tenant]({{<
  relref "../set-up-tenant" >}}).

## Cross-tenant queries
This section describes cross-tenant queries. 

<!-- ### Federation frontend

Tenant federation is handled by the `federation-frontend` service.
This service aggregates the data from multiple tenants in a single trace lookup or search query.

The `federation-frontend` runs with tenant federation enabled by default.
Queries with that contain the header `X-Org-Id` with multiple tenants separated by the `|` character,
are aggregated across all of specified tenants.
-->

#### Configuration

To run the federation frontend, configure the `target` option to be `federation-frontend`.
Then, you only need to indicate a proxy target to which the federation frontend will forward the queries.

```yaml
target: federation-frontend # Run the federation frontend only

federation:
  proxy_targets:
    - url: http://get-us-west/tempo
```

### Set up an access policy with tenant federation and a token

To allow queries to span both GET tenants, create a new access policy called `leadership`. 
For demonstration purposes, these tenants are named `team-engineering` and `team-finance`.
To create a raw access policy:

1. Create a new access policy `leadership`.
1Enable the `traces:read` scope.
1Add the tenants `team-engineering` and `team-finance`.
Alternatively, you can add the special tenant name `*` to create an access policy that has access to all tenants in the cluster.
1Create a new token for the access-policy and store the token in your
    clipboard:

{{< vimeo 681850303 >}}

### Set up a Grafana data source using the access policy

To set up a data source using the access policy:
1.  Create a new Tempo data source from the Grafana configuration menu.
1.  Enter the URL of your GET cluster, for example `http://enterprise-traces:3200`.
1.  From the **Auth** section, enable Basic auth.
1.  In the **User** field, enter: `team-engineering|team-finance` where all the names of the tenants that you want to query across are separated by the `|` pipe character.
1.  In the **Password** field, paste the token created in the token creation process.

Queries that are performed using this data source in either Explore or inside of
dashboards are performed across all the tenants that you specified in the **User**
field. 
These queries are processed as if all the data were in a single tenant.

To submit a query across all tenants that your access policy has access to, you can either:

1. Explicitly set the name of all the tenants separated by a pipe character "|" in the _username_.
For example, to query across `tenant1`, `tenant2`, and `tenant3` you would enter `tenant1|tenant2|tenant3`.
2. Set the username to a wildcard character `*`.
This will query all tenants that the access policy grants you access to, without requiring you to explicitly specify their names.

When using an access policy that has a wildcard (`*`) as the _username_,
you can query all tenants for that cluster by also specifying `*` as the _username_ in your data source URL.

Conversely, if you use a wildcard _username_ in your data source configuration with an access policy with specific tenants,
that data source has access to only those tenants.
