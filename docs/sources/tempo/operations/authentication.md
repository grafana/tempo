---
title: Manage authentication
menuTitle: Authentication
description: Describes how to add authentication to Grafana Tempo.
weight: 
---

# Manage authentication

Grafana Tempo does not come with any included authentication layer. You must run an authenticating reverse proxy in front of your services.

We recommend that in all [deployment modes](https://grafana.com/docs/tempo/<TEMPO_VERSION>/setup/deployment/) you add a reverse proxy to be deployed in front of Tempo, to direct client API requests to the various components.

A list of open-source reverse proxies you can use:

- [HAProxy](https://docs.haproxy.org/ )
- [NGINX](https://docs.nginx.com/nginx/) using their [guide on restricting access with HTTP basic authentication](https://docs.nginx.com/nginx/admin-guide/security-controls/configuring-http-basic-authentication/)
- [OAuth2 proxy](https://oauth2-proxy.github.io/oauth2-proxy/)
- [Pomerium](https://www.pomerium.com/docs), which has a [guide for securing Grafana](https://www.pomerium.com/docs/guides/grafana)

{{< admonition type="note" >}}
When using Tempo in multi-tenant mode, Tempo requires the HTTP header
`X-Scope-OrgID` to be set to a string identifying the tenant.
It's assumed that clients setting `X-Scope-OrgID` are trusted clients, and the responsibility of populating this value should be handled by the authenticating reverse proxy.
For more information, read the [multi-tenancy](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/manage-advanced-systems/multitenancy/) documentation.
{{< /admonition >}}
