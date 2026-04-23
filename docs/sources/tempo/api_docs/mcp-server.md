---
title: Tempo MCP server
description: Grafana Tempo exposes an MCP server to allow LLMs and AI assistants to interact with your trace data.
menuTitle: MCP server
weight: 800
---

# Model Context Protocol (MCP) Server

Tempo includes an MCP (Model Context Protocol) server that provides AI assistants and large language models (LLMs) with direct access to distributed tracing data through TraceQL queries and other endpoints.

For examples on how you can use the MCP server, refer to [LLM-powered insights into your tracing data: introducing MCP support in Grafana Cloud Traces](https://grafana.com/blog/2025/08/13/llm-powered-insights-into-your-tracing-data-introducing-mcp-support-in-grafana-cloud-traces/).

For more information on MCP, refer to the [MCP documentation](https://modelcontextprotocol.io/docs/getting-started/intro).

## Configuration

You can enable the MCP server in your Tempo configuration via YAML:

```yaml
query_frontend:
  mcp_server:
    enabled: true
```

Or via a command-line flag: `--query-frontend.mcp-server.enabled=true`.

{{< admonition type="warning" >}}
Be aware that using this feature may cause tracing data to be passed to an LLM or LLM provider. Consider the content of your tracing data and organizational policies when enabling this feature.
{{< /admonition >}}

The MCP server uses the same authentication and [multi-tenancy](../../operations/manage-advanced-systems/multitenancy/) behavior as other Tempo API endpoints.

## Available tools

The MCP server exposes the following tools that AI assistants can use to interact with your tracing data:

| Tool                      | Description                                                             |
| ------------------------- | ----------------------------------------------------------------------- |
| `traceql-search`          | Search for traces using TraceQL queries                                 |
| `traceql-metrics-instant` | Retrieve a single metric value given a TraceQL metrics query            |
| `traceql-metrics-range`   | Retrieve a metric series given a TraceQL metrics query                  |
| `get-trace`               | Retrieve a specific trace by ID                                         |
| `get-attribute-names`     | Get available attribute names for use in TraceQL queries                |
| `get-attribute-values`    | Get values for a specific scoped attribute name                         |
| `docs-traceql`            | Retrieve TraceQL documentation (basic, aggregates, structural, metrics) |

## Available resources

The MCP server also provides the following resources containing TraceQL documentation:

| Resource URI                | Description                                                 |
| --------------------------- | ----------------------------------------------------------- |
| `docs://traceql/basic`      | Basic TraceQL syntax, intrinsics, operators, and attributes |
| `docs://traceql/aggregates` | TraceQL aggregate functions (count, sum, etc.)              |
| `docs://traceql/structural` | Advanced structural query patterns                          |
| `docs://traceql/metrics`    | Generating metrics from tracing data with TraceQL           |

## Quick start

To experiment with the MCP server using dummy data and Claude Code:

1. Run the local docker-compose example in `/example/docker-compose/single-binary`. This exposes the MCP server at `http://localhost:3200/api/mcp`
1. Run `claude mcp add --transport=http tempo http://localhost:3200/api/mcp` to add a reference to Claude Code.
1. Run `claude` and ask some questions.

The Tempo MCP server uses the Streamable HTTP transport.
Any MCP client that supports this transport can connect directly using the URL `http://<tempo-host>:<port>/api/mcp`.
For example, in Cursor you can add the server with `type: "streamableHttp"` in your MCP configuration.

If your client doesn't support Streamable HTTP natively, you can use the [`mcp-remote`](https://www.npmjs.com/package/mcp-remote) package as a bridge.

![Claude Code interacting with the Tempo MCP server](/static/img/docs/tempo/claude-code-tempo-mcp.png)
