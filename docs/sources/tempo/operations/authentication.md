---
title: Manage authentication
menuTitle: Authentication
description: Describes how to add authentication to Grafana Tempo.
weight: 
---

# Manage authentication

Grafana Tempo does not come with any included authentication layer. You must run an authenticating reverse proxy in front of your services.

We recommend that in all [deployment modes](https://grafana.com/docs/tempo/<TEMPO_VERSION>/setup/deployment/) you add a reverse proxy in front of Tempo to direct client API requests to the various components.

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

## Protect internal and administrative endpoints

Tempo exposes operational endpoints on component HTTP and gRPC listeners for status pages, ring inspection, flush, shutdown, downscale workflows, and backend scheduling.
These endpoints are intended for internal operator access, automation, and component-to-component communication.
They aren't tenant-scoped API endpoints and shouldn't be exposed to tenants or untrusted clients.

Expose only the tenant-facing API routes that are required for ingestion and querying through public or tenant-facing ingress.
Keep component listeners, such as backend-scheduler and live-store, restricted to internal networks.

In monolithic deployments, or any deployment where a reverse proxy forwards a component listener, configure the proxy to block internal routes before forwarding requests to Tempo.
Depending on the enabled components, internal HTTP routes include:

- `/flush`
- `/shutdown`
- `/partition-ring`
- `/live-store/prepare-partition-downscale`
- `/live-store/prepare-downscale`
- `/status/backendscheduler`

The backend-scheduler also exposes gRPC methods for backend workers and redaction workflows.
Don't expose the backend-scheduler gRPC service to untrusted clients.
Restrict access to trusted backend workers and operators, including methods such as:

- `/tempopb.BackendScheduler/Next`
- `/tempopb.BackendScheduler/UpdateJob`
- `/tempopb.BackendScheduler/SubmitRedaction`

The `X-Scope-OrgID` header scopes tenant data.
It isn't an administrative authorization mechanism for cluster-wide operational endpoints.
