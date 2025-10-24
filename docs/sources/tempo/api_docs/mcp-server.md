---
title: Tempo MCP server
description: Grafana Tempo exposes an MCP server to allow LLMs and AI assistants to interact with your trace data.
menuTitle: MCP server
weight: 800
---

# Model Context Protocol (MCP) Server

Tempo includes an MCP (Model Context Protocol) server that provides AI assistants and Large Language Models (LLMs) with direct access to distributed tracing data through TraceQL queries and other endpoints.

For examples on how you can use the MCP server, refer to [LLM-powered insights into your tracing data: introducing MCP support in Grafana Cloud Traces](https://grafana.com/blog/2025/08/13/llm-powered-insights-into-your-tracing-data-introducing-mcp-support-in-grafana-cloud-traces/).

For more information on MCP, refer to the [MCP documentation](https://modelcontextprotocol.io/docs/getting-started/intro).

## Configuration

Enable the MCP server in your Tempo configuration:

```yaml
query_frontend:
  mcp_server:
    enabled: true
```

{{< admonition type="warning" >}}
Be aware that using this feature will likely cause tracing data to be passed to an LLM or LLM provider. Consider the content of your tracing data and organizational policies when enabling this.
{{< /admonition >}}

## Quick start

To experiment with the MCP server using dummy data and Claude Code:

1. Run the local docker-compose example in `/example/docker-compose/local`. This exposes the MCP server at `http://localhost:3200/api/mcp`
1. Run `claude mcp add --transport=http tempo http://localhost:3200/api/mcp` to add a reference to Claude Code.
1. Run `claude` and ask some questions.

This MCP server has also been tested successfully in cursor using the [`mcp-remote`](https://www.npmjs.com/package/mcp-remote) package.

![Claude Code interacting with the Tempo MCP server](/static/img/docs/tempo/claude-code-tempo-mcp.png)
