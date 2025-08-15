---
title: Tempo MCP Server
description: Grafana Tempo exposes an MCP Server to allow LLMs and AI assistants to interact with your trace data
menuTitle: MCP Server
weight: 800
---

# Model Context Protocol (MCP) Server

Tempo includes an MCP (Model Context Protocol) server that provides AI assistants and Large Language Models with direct access to distributed tracing data through TraceQL queries and other endpoints.

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

1. Run the local docker-compose example in `/example/docker-compose/local`. This will expose the mcp server at http://localhost:3200/api/mcp
1. Run `claude mcp add --transport=http tempo http://localhost:3200/api/mcp` to add a reference to Claude Code.
1. Run `claude` and ask some questions

This MCP server has also been tested succesfully in cursor using the [`mcp-remote`](https://www.npmjs.com/package/mcp-remote) package.

![Claude Code interacting with the Tempo MCP server](/static/img/docs/tempo/claude-code-tempo-mcp.png)