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

Multi-tenancy lets multiple teams or customers share a Tempo cluster while keeping their trace data separate.
A tenant is a group of users or services whose traces are stored and queried together.
Tempo uses the `X-Scope-OrgID` header to enforce multi-tenancy in Tempo and Grafana Enterprise Traces.
It is set to the tenant (or "organization") name.
It is used for scoped writes (ingest) so that each span is stored under its specified tenant, and scoped reads so that queries return only that tenant's data.

For a working configuration, consult the [multi-tenant example](https://github.com/grafana/tempo/tree/main/example/docker-compose/multitenant)
in the repository.

## When to use multi-tenancy

### Separate teams within an organization

A platform team can run one Tempo cluster for the payments and checkout teams instead of maintaining a cluster for each team.
Use a tenant ID such as `team-payments` for the payments team's traces and `team-checkout` for the checkout team's traces.

Configure each team's collector to send its tenant ID in the `X-Scope-OrgID` header when exporting traces.
Configure each team's Grafana Tempo data source to send the same tenant ID when querying.
For example, both the payments collector and the payments data source use `X-Scope-OrgID: team-payments`.
Queries from that data source return the payments tenant's traces.

Choose tenant boundaries to match how you troubleshoot services.
Services whose spans need to be queried together as one trace should send those spans to the same tenant.
For queries across several tenants, refer to [Cross-tenant query federation](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/manage-advanced-systems/cross_tenant_query/).

### Separate customers of a service

A SaaS provider can use a tenant for each customer, such as `customer-acme` and `customer-contoso`.
The provider can operate one Tempo cluster while keeping each customer's traces separate.
Send Acme's traces with `X-Scope-OrgID: customer-acme` and use the same header for Acme's queries.
Use `customer-contoso` for Contoso's ingestion and queries.

Tempo doesn't include an authentication layer.
Use an authenticating reverse proxy to set the tenant header based on the customer's identity and permissions, rather than accepting a tenant ID chosen by the customer.
For more information, refer to [Manage authentication](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/authentication/).

For either use case, [per-tenant settings](#configure-per-tenant-settings) let you set different ingestion limits or block retention periods for each tenant.
Enable multi-tenancy on Tempo and configure tenant headers as described in the following sections.

## Tenant IDs

Tenant IDs are transmitted to Tempo via the `X-Scope-OrgID` HTTP header. This header must be included in all requests to Tempo when multi-tenancy is enabled.

Multi-tenancy on ingestion is supported with both gRPC and HTTP for OTLP (OpenTelemetry Protocol). You can add the header in:

- OpenTelemetry Collector configuration
- Grafana Alloy configuration
- Any HTTP/gRPC client using `curl` or other relevant tools

## Example Alloy configuration

The following example shows how to configure Grafana Alloy to send traces with a tenant ID.

```
otelcol.exporter.otlphttp "tempo" {
    // Define the client for exporting.
    client {
        // Send the X-Scope-OrgID header to the Tempo instance for multi-tenancy (tenant 1234).
        headers = {
            "X-Scope-OrgID" = "1234",
        }

        // Send to the locally running Tempo instance, on port 4318 (OTLP HTTP).
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

1. Configure the OpenTelemetry Collector to attach the `X-Scope-OrgID` header on push:

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
       httpHeaderName1: "X-Scope-OrgID"
     secureJsonData:
       httpHeaderValue1: "foo-bar-baz"
   ```

1. Enable multi-tenancy on the Tempo backend by setting the following configuration value on all Tempo components:

   ```yaml
   multitenancy_enabled: true
   ```

   or from the command line:

   ```shell
   tempo --multitenancy.enabled=true
   ```

   This option forces all Tempo components to require the `X-Scope-OrgID` header.

## Configure per-tenant settings

After you enable multi-tenancy, you can customize settings on a per-tenant basis using runtime overrides.
This is useful when different tenants have different requirements, for example varying block retention periods or ingestion limits.

This configuration works for both Tempo 2.x and Tempo 3.x architectures.
If you are upgrading from Tempo 2.x, refer to the [Tempo 3.0 migration guide](https://grafana.com/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/migrate-to-3/).

To configure per-tenant overrides, create a separate overrides file and reference it from your main Tempo configuration:

```yaml
# tempo.yaml
overrides:
  per_tenant_override_config: /conf/overrides.yaml
```

Define tenant-specific settings in the `overrides.yaml` file.
The following example sets different block retention periods and ingestion limits for two tenants:

```yaml
# overrides.yaml
overrides:
  "tenant-a":
    compaction:
      block_retention: 24h
    ingestion:
      burst_size_bytes: 20000000
      rate_limit_bytes: 4000000
      max_traces_per_user: 10000
    global:
      max_bytes_per_trace: 30000000
    metrics_generator:
      collection_interval: 15s
      ring_size: 2
      max_active_series: 5000
      processors:
        - span-metrics
        - service-graphs
      forwarder:
        queue_size: 5000
        workers: 3
  "tenant-b":
    compaction:
      block_retention: 48h
    ingestion:
      burst_size_bytes: 20000000
      rate_limit_bytes: 4000000
      max_traces_per_user: 10000
    global:
      max_bytes_per_trace: 30000000
    metrics_generator:
      ring_size: 2
      max_active_series: 100000
      processors: []
      forwarder:
        queue_size: 5000
        workers: 3
```

{{< admonition type="warning" >}}
When using per-tenant overrides, you must set every field you want to preserve.
Tempo replaces unset fields with zero values, not with the defaults from your main configuration.
Omitting a field can lead to unexpected behavior, such as a `0s` retention effectively disabling block retention for that tenant.
{{< /admonition >}}

The overrides file is dynamically loaded, so you can update it at runtime without restarting Tempo.

For the full list of available override fields, refer to [Overrides](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#overrides).
