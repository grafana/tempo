---
title: Grafana Tempo Datasource
description: Use the Tempo Operator to deploy Tempo and use it as a data source with Grafana
aliases:
 - /docs/tempo/latest/gateway
 - /docs/tempo/operations/gateway
weight: 15
---

# Grafana Tempo Datasource

Using the instructions on this page, you can configure the `TempoStack` to send data to Grafana and configure the Tempo data source.  

You can choose to either use Tempo Operator's gateway or not: 

* If the `TempoStack` is deployed using the gateway, you'll need to provide authentication information  to Grafana, along with the URL of the tenant it wants to access.

* If the gateway is not used, then you need to make sure Grafana can access to the `frontend-query` endpoints.

## Use with gateway

The gateway, an optional component deployed as part of Tempo Operator, provides secure access to Tempo's distributor (for example, for pushing spans) and query-frontend (for example, for querying traces) via consulting an OAuth/OIDC endpoint for the request subject.

The OIDC configuration expects `clientID`, `clientSecret` which should be provided via a Kubernetes secret that the `TempoStack` admin provides upfront.

The gateway exposes all Tempo query endpoints, so you can use the endpoint as a Tempo Grafana data source.

If Grafana is configured with some OAuth provider, such as generic OAuth, the `TempoStack` with the gateway should be deployed using the same `clientID` and `clientSecret`:

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

Then deploy `TempoStack` with gateway enabled:

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

To use it as a data source, set the Authentication Method to **Forward Oauth Identify** using the same `clientID` and `clientSecret` for gateway and for the OAuth configuration. This will forward the `access_token` to the gateway so it can authenticate the client.

<p align="center"><img src="../grafana_datasource_tempo.png" alt="Tempo Datasource configured for the gateway forwarding oauth access token"></p>

If you prefer to set the Bearer token directly and not use the  **Forward Oauth Identify**, you can add it to the "Authorization" Header.

<p align="center"><img src="../grafana_datasource_tempo_headers.png" alt="Tempo Datasource configured for the gateway using Bearer token"></p>

## Without the gateway

If you are not using the gateway, make sure your Grafana can access to the query-frontend endpoints, you can do this by creating an ingress or a route in OpenShift.

Once you have the endpoint, you can set it as `URL` when you create the Tempo data source.