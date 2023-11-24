---
title: Grafana Tempo Datasource
description: Use the Tempo Operator to deploy Tempo and use it as a data source with Grafana
aliases:
 - /docs/tempo/latest/gateway
 - /docs/tempo/operations/gateway
weight: 15
---

# Grafana Tempo Datasource

There are one main thing to take into account when you deploy the `TempoStack` and want to use as Grafana datasource.

If the tempo is deployed the Gateway in front, an authentication information needs to be provided to Grafana, and the URL of the tenant it wants to access.

If the gateway is not used, the only requirement is to Grafana can access to the `frontend-query` endpoints.

## Use with gateway

The gateway is an optional component deployed as part of Tempo Operator. It provides secure access to Tempo's distributor (i.e. for pushing spans) and query-frontend (i.e. for querying traces) via consulting an OAuth/OIDC endpoint for the request subject.

The OIDC configuration expects `clientID`, `clientSecret`  which should be provided via a Kubernetes secret that the TempoStack admin provides upfront.

The gateway exposes all tempo query endpoints, thus we can use the endpoint as a Tempo Grafana Datasource.

If  Grafana is configured with some OAuth provider, e.g. generic oauth, TempoStack with the gateway should be deployed  using the same `clientID` and `clientSecret`:

```yaml
apiVersion: v1
kind: Secret
metadata:
 name: oidc-test
stringData:
 clientID: <clientID used for grafana authentication>
 clientSecret: <clientSecret used for grafana authentication>
type: Opaque
```

Then deploy TempoStack with gateway enabled

```yaml
spec:
 template:
  gateway:
   enabled: true
 tenants:
  mode: static
  authentication:
    - tenantName: test-oidc
      tenantId: test-oidc
      oidc:
      issuerURL: http://dex:30556/dex
      redirectURL: http://tempo-foo-gateway:8080/oidc/test-oidc/callback
      usernameClaim: email
      secret:
       name: oidc-test
```

In the url you should set it to `http://<HOST>:<PORT>/api/traces/v1/{tenant}/tempo/`

In order to use it as a data source you need to set the Authentication Method to **Forward Oauth Identify**, and make sure  you use the same `clientID` and `clientSecret` for gateway and for the oauth configuration. This will forward the access_token to the gateway ,so it can authenticate the client.

<p align="center"><img src="../grafana_datasource_tempo.png" alt="Tempo Datasource configured for the gateway forwarding oauth access token"></p>

If you prefer to set the Bearer token directly, and not use the  **Forward Oauth Identify** you can add it to the "Authorization" Header 

<p align="center"><img src="../grafana_datasource_tempo_headers.png" alt="Tempo Datasource configured for the gateway using Bearer token"></p>

## Without the gateway

If you are not using the gateway make sure your grafana can access to the query-frontend endpoints, you can do this 
creating an ingress or a route in OpenShift.

Once you have the endpoint you can set it as `URL` when you create the tempo datasource.