---
title: Federated query frontend
menuTitle: Federation
description: Learn how to query traces across multiple Tempo instances using the federated query frontend.
weight: 650
---

# Federated query frontend

{{< admonition type="warning" >}}
The federated query frontend is an **experimental, community-driven feature**.
It's partially implemented and under active development.
The API and configuration may change in future releases.
Use in production at your own risk.
{{< /admonition >}}

The federated query frontend is a new Tempo component that enables querying traces across multiple independent Tempo instances from a single endpoint.
This is useful for multi-region, multi-cluster, or multi-tenant deployments where traces are distributed across separate Tempo backends.

## Feature status

The following table shows the current implementation status of federation features:

| Feature | Status | Description |
| ------- | ------ | ----------- |
| Trace by ID (`/api/traces/{traceID}`) | ✅ Implemented | Query a trace by ID across all instances |
| Trace by ID v2 (`/api/v2/traces/{traceID}`) | ✅ Implemented | Query a trace by ID with v2 API |
| Search (`/api/search`) | ❌ Not implemented | Search for traces across instances |
| Search tags (`/api/search/tags`) | ❌ Not implemented | Query available tags across instances |
| Search tag values (`/api/search/tag/{tag}/values`) | ❌ Not implemented | Query tag values across instances |
| TraceQL metrics (`/api/metrics/*`) | ❌ Not implemented | Metrics queries across instances |
| Streaming gRPC | ❌ Not supported | Streaming endpoints aren't supported |
| Instance health checking | ❌ Not implemented | Monitor and route based on instance health |
| Weighted routing | ❌ Not implemented | Route requests based on instance capacity |
| Caching | ❌ Not implemented | Cache responses from federated queries |

## Use cases

Federation addresses common scenarios in large-scale distributed systems:

- **Multi-region deployments**: Services in different regions send traces to local Tempo instances
- **Multi-cluster setups**: Each Kubernetes cluster has its own Tempo instance
- **Organizational boundaries**: Different teams operate independent Tempo instances

Without federation, querying a trace ID requires knowing which instance contains it, or querying each instance manually.
Federation solves this by providing a unified query layer.

## Architecture

The federated query frontend acts as a proxy that fans out requests to multiple Tempo instances and combines the results.

When you query a trace ID, the federated frontend:

1. Sends the request to all configured Tempo instances in parallel
1. Collects responses from each instance
1. Combines the trace spans from all instances into a single response
1. Returns the merged trace to the client

## Configuration

Configure the federated query frontend using the `federated_frontend` configuration block.

You must set the `target` to `federated-query-frontend` as a command-line argument when starting Tempo.
This starts Tempo as a dedicated federated query frontend instead of the standard all-in-one or microservices mode.

```sh
tempo -target=federated-query-frontend -config.file=/etc/tempo.yaml
```

### Basic configuration

The following example shows a minimal federation configuration:

```yaml
server:
  http_listen_port: 3200

federated_frontend:
  concurrent_requests: 4
  instances:
    - name: "tempo-1"
      endpoint: "http://tempo-1:3200"
    - name: "tempo-2"
      endpoint: "http://tempo-2:3200"
```

### Configuration options

The following table describes the available configuration options:

| Option | Summary | Required | Type | Default |
| ------ | ------- | -------- | ---- | ------- |
| `concurrent_requests` | Maximum number of concurrent requests to send to instances | No | Integer | `4` |
| `instances` | List of Tempo instances to query | Yes | List | N/A |

### Instance configuration

Each instance in the `instances` list supports the following options:

| Option | Summary | Required | Type | Default |
| ------ | ------- | -------- | ---- | ------- |
| `name` | Friendly name for this instance, used in logs and metrics | Yes | String | N/A |
| `endpoint` | Base URL for this Tempo instance | Yes | String | N/A |
| `timeout` | Request timeout for this instance | No | Duration | `30s` |
| `org_id` | Tenant ID to use for this instance, overrides the request tenant | No | String | N/A |
| `headers` | Additional headers to send with requests | No | Map | N/A |

### Advanced configuration

The following example shows a complete configuration with all options:

```yaml
server:
  http_listen_port: 3200

federated_frontend:
  concurrent_requests: 4
  instances:
    - name: "tempo-region-1"
      endpoint: "http://tempo-1.region-1.svc:3200"
      timeout: 30s
      org_id: "tenant-1"
      headers:
        X-Custom-Header: "value"
    - name: "tempo-region-2"
      endpoint: "http://tempo-2.region-2.svc:3200"
      timeout: 45s
    - name: "tempo-region-3"
      endpoint: "https://tempo-3.example.com"
      timeout: 60s
      headers:
        Authorization: "Bearer token"
```

## Contributing

The federated query frontend is a community-driven feature.
Contributions are welcome to help expand its capabilities.

Refer to the [Tempo GitHub repository](https://github.com/grafana/tempo) to:

- Report issues or request features
- Submit pull requests to implement missing functionality
- Discuss design decisions in GitHub issues

## Related resources

- [Tempo configuration reference](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/)
- [Multi-tenancy](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/manage-advanced-systems/multitenancy/)
- [Deploy Tempo](https://grafana.com/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/deploy/)
