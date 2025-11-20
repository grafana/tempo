---
aliases:
  - ../configuring/about-tenant-ids/
description: Learn about tenant IDs.
menuTitle: Tenant IDs
title: Tenant IDs
weight: 40
---

# Tenant IDs

Within a Grafana Tempo cluster, the tenant ID is the unique identifier of a tenant. Tenant IDs enable multi-tenancy in Tempo, allowing multiple organizations or tenants to share the same Tempo cluster while maintaining data isolation.

Tempo uses the `X-Scope-OrgID` HTTP header to identify and enforce tenant boundaries. This header is set to the tenant ID value and is used for:

- **Scoped writes (ingest)**: Each span is stored under its specified tenant, ensuring data isolation at the storage level
- **Scoped reads (queries)**: Queries return only data belonging to the specified tenant

For more information about setting up multi-tenancy, refer to [Enable multi-tenancy](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/manage-advanced-systems/multitenancy/).

## Characters and length restrictions

Tenant IDs must be less-than or equal-to 150 bytes in length. The length is measured in bytes, not characters, which is important for multi-byte characters.

Tenant IDs can only include the following supported characters:

- Alphanumeric characters (`0-9`, `a-z`, `A-Z`)
- Special characters (`!`, `-`, `_`, `.`, `*`, `'`, `(`, `)`)

All other characters, including slashes (`/`), backslashes (`\`), and whitespace, aren't supported.

Tenant IDs are case-sensitive. For example, `Tenant-123` and `tenant-123` are treated as different tenant IDs.

The tenant ID should not be empty.

For security reasons, `.` and `..` aren't valid tenant IDs. These values are restricted to prevent path traversal attacks.


## Cross-tenant queries

Cross-tenant queries are supported for search, search-tags, and trace-by-ID search operations.
Refer to [Cross-tenant query federation](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/manage-advanced-systems/cross_tenant_query/) for more information.

## Multitenancy

Tenant IDs are transmitted to Tempo via the `X-Scope-OrgID` HTTP header. This header must be included in all requests to Tempo when multi-tenancy is enabled.

Multi-tenancy on ingestion is supported with both gRPC and HTTP for OTLP (OpenTelemetry Protocol). You can add the header in:

- OpenTelemetry Collector configuration
- Grafana Alloy configuration
- Any HTTP/gRPC client using `curl` or other relevant tools

Refer to [Enable multi-tenancy](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/manage-advanced-systems/multitenancy/) for more information.
