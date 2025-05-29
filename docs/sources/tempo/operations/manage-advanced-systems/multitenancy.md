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

Tempo is a multi-tenant distributed tracing backend. 
Tempo uses the `X-Scope-OrgID` header to enforce multi-tenancy in Tempo and Grafana Enterprise Traces.
It is set to the tenant (or “organization”) name.
It is used for scoped writes (ingest) so that each span is stored under its specified tenant, and scoped reads so that queries return only that tenant’s data.

If you're interested in setting up multi-tenancy, consult the [multi-tenant example](https://github.com/grafana/tempo/tree/main/example/docker-compose/otel-collector-multitenant)
in the repository. This example uses the following settings to achieve multi-tenancy in Tempo.

Multi-tenancy on ingestion is supported with GPRC and HTTP for OTLP.
You can add headers both in the OpenTelemetry Collector and Grafana Alloy or using `curl` or any other relevant HTTP/gRPC protocol tool. 
Here is an example configuration for Alloy. 

```
otelcol.exporter.otlphttp "tempo" {
    // Define the client for exporting.
    client {
        // Send the X-Scope-OrgID header to the Tempo instance for multi-tenancy (tenant 1234).
        headers = {
            "X-Scope-OrgID" = "1234",
        }

        // Send to the locally running Tempo instance, on port 4317 (OTLP gRPC).
        endpoint = "http://tempo:4318"

        // Configure TLS settings for communicating with the endpoint.
        tls {
            // The connection is insecure.
            insecure = true
            // Do not verify TLS certificates when connecting.
            insecure_skip_verify = true
        }
    }
}
```

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
