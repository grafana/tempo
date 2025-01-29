---
title: Enable multi-tenancy
menuTitle: Enable multi-tenancy
weight: 100
description: Enable multi-tenancy in Tempo using the X-Scope-OrgID header.
aliases:
  - ../../configuration/multitenancy/ # https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/multitenancy/
  - ../multitenancy/ # https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/multitenancy/
---

# Enable multi-tenancy

Tempo is a multi-tenant distributed tracing backend. It supports multi-tenancy through the use
of a header: `X-Scope-OrgID`.

If you're interested in setting up multi-tenancy, consult the [multi-tenant example](https://github.com/grafana/tempo/tree/main/example/docker-compose/otel-collector-multitenant)
in the repository. This example uses the following settings to achieve multi-tenancy in Tempo.

{{< admonition type="note" >}}
Multi-tenancy on ingestion is currently [only working](https://github.com/grafana/tempo/issues/495) with GPRC and this may never change. It's strongly recommended to use the OpenTelemetry Collector to support multi-tenancy.
{{< /admonition >}}

## Configure multi-tenancy

1. Configure the OTEL Collector to attach the `X-Scope-OrgID` header on push:

   ```
   exporters:
     otlp:
       headers:
         x-scope-orgid: foo-bar-baz
   ```

1. Configure the Tempo data source in Grafana to pass the tenant with the same header:

   ```yaml
   - name: Tempo-Multitenant
     jsonData:
       httpHeaderName1: 'X-Scope-OrgID'
     secureJsonData:
       httpHeaderValue1: 'foo-bar-baz'
   ```

1. Enable multi-tenancy on the Tempo backend by setting the following configuration value on all Tempo components:

   ```yaml
   multitenancy_enabled: true
   ```

   or from the command line:

   ```yaml
   --multitenancy.enabled=true
   ```

   This option forces all Tempo components to require the `X-Scope-OrgID` header.
