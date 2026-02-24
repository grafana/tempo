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
| `--mem-ballast-size-mbs` | Size of memory ballast to allocate in MBs | `0` |
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
  In Tempo 3.0, the `ingester`, `compactor`, and `scalable-single-binary` targets were removed as as part of the [Project Rhythm architecture](/docs/tempo/<TEMPO_VERSION>/introduction/architecture/#project-rhythm).
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

## Logging settings

| Flag | Description | Default |
| --- | --- | --- |
| `--log.level` | Only log messages with the given severity or above. Valid levels: `debug`, `info`, `warn`, `error` | `info` |
| `--log.format` | Output log messages in the given format. Valid formats: `logfmt`, `json` | `logfmt` |

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

## Module configuration

You can use additional flags to configure individual Tempo modules, such as the distributor, block-builder, live-store, querier, backend-scheduler, backend-worker, and their components.
These flags follow a pattern like `--<module>.<setting>` and are extensively documented in the configuration file format.

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

