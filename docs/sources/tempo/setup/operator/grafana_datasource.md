---
title: Grafana data source
description: Use the Tempo Operator to deploy Tempo and use it as a data source with Grafana
aliases:
 - /docs/tempo/operator/grafana_datasource
weight: 400
---

# Grafana data source

You can use Grafana to query and visualize traces of the `TempoStack` instance by configuring a Tempo data source in Grafana.

## Use Grafana Operator

If your Grafana instance is managed by the [Grafana Operator](/docs/grafana-cloud/developer-resources/infrastructure-as-code/grafana-operator/), you can instruct the Tempo Operator to create a data source (`GrafanaDatasource` custom resource):

```yaml
apiVersion: tempo.grafana.com/v1alpha1
kind: TempoStack
spec:
  observability:
    grafana:
      createDatasource: true
```

{{< admonition type="note" >}}
The feature gate `featureGates.grafanaOperator` must be enabled in the Tempo Operator configuration.
{{< /admonition >}}

## Manual data source configuration

You can choose to either use Tempo Operator's gateway or not: 

* If the `TempoStack` is deployed using the gateway, you'll need to provide authentication information to Grafana, along with the URL of the tenant from which you expect to see the traces.

* If the gateway is not used, then you need to make sure Grafana can access the `query-frontend` endpoints.

For more information, refer to the [Tempo data source for Grafana](/docs/grafana/latest/datasources/tempo/).

### Use with gateway

The gateway, an optional component deployed as part of Tempo Operator, provides secure access to Tempo's distributor (for example, for pushing spans) and query-frontend (for example, for querying traces) via consulting an OAuth/OIDC endpoint for the request subject.

The OIDC configuration expects `clientID` and `clientSecret`. They should be provided via a Kubernetes secret that the `TempoStack` admin provides upfront.

The gateway exposes all Tempo query endpoints, so you can use the endpoint as a Tempo data source for Grafana.

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

Set the data source URL parameter to `http://<HOST>:<PORT>/api/traces/v1/{tenant}/tempo/`, where `{tenant}` is the name of the tenant.

To use it as a data source, set the Authentication Method to **Forward Oauth Identify** using the same `clientID` and `clientSecret` for gateway and for the OAuth configuration. This will forward the `access_token` to the gateway so it can authenticate the client.

<p align="center"><img src="../grafana_datasource_tempo.png" alt="Tempo data source configured for the gateway forwarding OAuth access token"></p>

If you prefer to set the Bearer token directly and not use the  **Forward Oauth Identify**, you can add it to the "Authorization" Header.

<p align="center"><img src="../grafana_datasource_tempo_headers.png" alt="Tempo data source configured for the gateway using Bearer token"></p>

### Without the gateway

If you are not using the gateway, make sure your Grafana can access to the query-frontend endpoints, you can do this by creating an ingress or a route in OpenShift.

Once you have the endpoint, you can set it as `URL` when you create the Tempo data source.
