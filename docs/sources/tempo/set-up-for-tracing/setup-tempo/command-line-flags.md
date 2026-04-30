---
title: Command line flags
menuTitle: Command line flags
description: Reference for Tempo command line flags
weight: 900
aliases:
  - ../../setup/command-line-flags/ # /docs/tempo/next/setup/command-line-flags/
---

# Command line flags

Tempo provides various command-line flags to configure its behavior when starting the binary. This document serves as a reference for these flags.

## Global flags

| Flag | Description | Default |
| --- | --- | --- |
| `--version` | Print this build's version information and exit | `false` |
| `--mutex-profile-fraction` | Override default mutex profiling fraction | `0` |
| `--block-profile-threshold` | Override default block profiling threshold | `0` |
| `--config.file` | Configuration file to load | |
| `--config.expand-env` | Whether to expand environment variables in config file | `false` |
| `--config.verify` | Verify configuration and exit | `false` |

## Target flag

The deployment mode is determined by the runtime configuration `target`, or
by using the `-target` flag on the command line.
The default target is `all`, which runs all components in a single process (monolithic deployment mode).

| Flag | Description | Default |
| --- | --- | --- |
| `--target` | Target module to run | `all` |

Valid target values:

| Target | Description |
| --- | --- |
| `all` | Monolithic mode. Runs all components in a single process. |
| `distributor` | Receives and distributes trace data to downstream components. |
| `metrics-generator` | Generates metrics from ingested trace data. |
| `querier` | Queries the backend storage for traces and metrics. |
| `query-frontend` | Provides search API and splits queries for parallelism. |
| `block-builder` | Consumes data from Kafka and writes blocks to backend storage. |
| `backend-scheduler` | Schedules and coordinates backend query jobs across workers. |
| `backend-worker` | Executes query jobs assigned by the backend scheduler. |
| `live-store` | Serves recently ingested data from Kafka for real-time queries. |

{{< admonition type="note" >}}
  In Tempo 3.0, the `ingester`, `compactor`, and `scalable-single-binary` targets were removed as part of the new [Tempo architecture](/docs/tempo/<TEMPO_VERSION>/introduction/architecture/).
  {{< /admonition >}}

Refer to the [Plan your Tempo deployment](../plan/) documentation for information on deployment modes.

## Authentication and multitenancy

| Flag | Description | Default |
| --- | --- | --- |
| `--multitenancy.enabled` | Set to true to enable multitenancy | `false` |
| `--auth.enabled` | **Deprecated. Use `--multitenancy.enabled` instead.** Set to true to enable auth. This flag will be removed in a future release. | `false` |

## HTTP and API settings

| Flag | Description | Default |
| --- | --- | --- |
| `--http-api-prefix` | String prefix for all HTTP API endpoints | `""` |
| `--enable-go-runtime-metrics` | Set to true to enable all Go runtime metrics | `false` |
| `--shutdown-delay` | How long to wait between SIGTERM and shutdown. After receiving SIGTERM, Tempo reports not-ready status via the `/ready` endpoint. | `0` |

## Span profiling

| Flag | Description | Default |
| --- | --- | --- |
| `--span-profiling` | Enable span profiling via `otelpyroscope`. When enabled, Tempo attaches pprof goroutine labels (`span_id`, `span_name`) to OTel spans and adds a `pyroscope.profile.id` attribute to root spans, enabling profile-to-trace correlation in Pyroscope. Requires an OTLP exporter to be configured through environment variables (`OTEL_TRACES_EXPORTER`, `OTEL_EXPORTER_OTLP_ENDPOINT`, or `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT`). | `false` |

You can also set this option in the configuration file:

```yaml
span_profiling: true
```

## Health check

| Flag | Description | Default |
| --- | --- | --- |
| `--health` | Run a health check against the `/ready` endpoint and exit. Returns exit code `0` if healthy, `1` if unhealthy. | `false` |
| `--health.url` | URL to check when running a health check | `http://localhost:3200/ready` |

The Tempo container image uses a [distroless base image](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/upgrade/#busybox-removed-from-tempo-image) that doesn't include a shell, `curl`, `wget`, or other utilities. This means the common Docker health check pattern `HEALTHCHECK CMD curl -f http://localhost:3200/ready` doesn't work.

The `--health` flag provides a native alternative. It doesn't require a Tempo configuration file, so it can be used directly in a `HEALTHCHECK` instruction:

```dockerfile
HEALTHCHECK CMD ["/tempo", "--health"]
```

Kubernetes users typically don't need this flag because they can configure `httpGet` readiness and liveness probes directly against the [`/ready` endpoint](/docs/tempo/<TEMPO_VERSION>/api_docs/#readiness-probe).

## Logging settings

| Flag | Description | Default |
| --- | --- | --- |
| `--log.level` | Only log messages with the given severity or above. Valid levels: `debug`, `info`, `warn`, `error` | `info` |
| `--log.format` | Output format for log messages. Valid formats: `logfmt`, `json` | `logfmt` |

## Server settings

| Flag | Description | Default |
| --- | --- | --- |
| `--server.http-listen-port` | HTTP server listen port | `3200` |
| `--server.grpc-listen-port` | gRPC server listen port | `9095` |

## Memberlist settings

| Flag | Description | Default |
| --- | --- | --- |
| `--memberlist.host-port` | Host port to connect to memberlist cluster | |
| `--memberlist.bind-port` | Port for memberlist to communicate on | `7946` |
| `--memberlist.message-history-buffer-bytes` | Size in bytes for the message history buffer | `0` |

## MCP server

| Flag | Description | Default |
| --- | --- | --- |
| `--query-frontend.mcp-server.enabled` | Set to true to enable the MCP server | `false` |

Tempo includes an [MCP (Model Context Protocol)](https://modelcontextprotocol.io/docs/getting-started/intro) server that provides AI assistants and Large Language Models (LLMs) with direct access to distributed tracing data through TraceQL queries and other endpoints.

Refer to the [Model Context Protocol (MCP) Server documentation](https://grafana.com/docs/tempo/<TEMPO_VERSION>/api_docs/mcp-server/) for more information.

## Module configuration

You can use additional flags to configure individual Tempo modules, such as the distributor, block-builder, live-store, querier, backend-scheduler, backend-worker, and their components.
These flags follow a pattern like `--<module>.<setting>` and are documented in the [configuration file format](/docs/tempo/<TEMPO_VERSION>/configuration/).

Use the configuration file approach described in the [Configuration documentation](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/).
The documentation has a comprehensive list of all configuration options.

## Usage examples

Start Tempo with a configuration file:

```bash
tempo --config.file=/etc/tempo/config.yaml
```

Start Tempo with a specific target:

```bash
tempo --target=distributor --config.file=/etc/tempo/config.yaml
```

Verify configuration without starting Tempo:

```bash
tempo --config.file=/etc/tempo/config.yaml --config.verify
```

Print version information:

```bash
tempo --version
```

Start Tempo with debug-level logging for troubleshooting:

```bash
tempo --config.file=/etc/tempo/config.yaml --log.level=debug
```

Use environment variable expansion in the config file, which lets you inject secrets or environment-specific values at startup:

```bash
tempo --config.file=/etc/tempo/config.yaml --config.expand-env
```

Start the distributor component on a custom HTTP port with JSON-formatted logs, typical for a microservices deployment behind a load balancer:

```bash
tempo --target=distributor \
  --config.file=/etc/tempo/config.yaml \
  --server.http-listen-port=3200 \
  --log.format=json
```

Start Tempo in monolithic mode with multitenancy enabled and a graceful shutdown delay of 30 seconds, allowing in-flight requests to complete during rolling updates:

```bash
tempo --config.file=/etc/tempo/config.yaml \
  --multitenancy.enabled \
  --shutdown-delay=30s
```

Run a health check against a custom URL:

```bash
tempo --health --health.url=http://localhost:3200/ready
```

