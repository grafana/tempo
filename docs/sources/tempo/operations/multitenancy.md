---
title: Enable multi-tenancy
menuTitle: Enable multi-tenancy
weight: 60
description: Enable multi-tenancy in Tempo using the X-Scope-OrgID header.
aliases:
- /docs/tempo/configuratioon/multitenancy
- /docs/tempo/operations/multitenancy
---

# Enable multi-tenancy

Tempo is a multi-tenant distributed tracing backend. It supports multi-tenancy through the use
of a header: `X-Scope-OrgID`.

If you're interested in setting up multi-tenancy, consult the [multi-tenant example](https://github.com/grafana/tempo/tree/main/example/docker-compose/otel-collector-multitenant)
in the repository. This example uses the following settings to achieve multi-tenancy in Tempo.

{{< admonition type="note" >}}
Multi-tenancy on ingestion is currently [only working](https://github.com/grafana/tempo/issues/495) with GPRC and this may never change. It's strongly recommended to use the OpenTelemetry Collector to support multi-tenancy.
{{% /admonition %}}

## Configure multi-tenancy

1. Configure the OTEL Collector to attach the `X-Scope-OrgID` header on push:

   ```
   exporters:
     otlp:
       headers:
         x-scope-orgid: foo-bar-baz
   ```

1. Configure the Tempo data source in Grafana to pass the tenant with the same header:

   ```
   - name: Tempo-Multitenant
     jsonData:
       httpHeaderName1: 'X-Scope-OrgID'
     secureJsonData:
       httpHeaderValue1: 'foo-bar-baz'
   ```

1. Enable multi-tenancy on the Tempo backend by setting the following configuration value on all Tempo components:

   ```
   multitenancy_enabled: true
   ```

   or from the command line:

   ```
   --multitenancy.enabled=true
   ```

   This option forces all Tempo components to require the `X-Scope-OrgID` header.

<!-- Commented out since 7.4 is no longer supported.
### Grafana 7.4.x

Grafana 7.4.x has the following configuration requirements:

- Configure the Tempo data source in Grafana to pass the tenant as a bearer token. This is necessary because it is the only header that Jaeger can be configured to pass to its GRPC plugin.

```
- name: Tempo-Multitenant
  jsonData:
    httpHeaderName1: 'Authorization'
  secureJsonData:
    httpHeaderValue1: 'Bearer foo-bar-baz'
```

- Configure Jaeger Query to pass the bearer token to its backend.

```
--query.bearer-token-propagation=true
```
-->


