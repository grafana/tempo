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
The default target is `all`, which is the monolithic deployment mode.

| Flag | Description | Default |
| --- | --- | --- |
| `--target` | Target module to run | `all` |

Refer to the [Plan your Tempo deployment](../plan/) documentation for information on deployment modes.

## Authentication and multitenancy

| Flag | Description | Default |
| --- | --- | --- |
| `--auth.enabled` | Set to true to enable auth (deprecated: use multitenancy.enabled) | `false` |
| `--multitenancy.enabled` | Set to true to enable multitenancy | `false` |

## HTTP and API settings

| Flag | Description | Default |
| --- | --- | --- |
| `--http-api-prefix` | String prefix for all HTTP API endpoints | `""` |
| `--enable-go-runtime-metrics` | Set to true to enable all Go runtime metrics | `false` |
| `--shutdown-delay` | How long to wait between SIGTERM and shutdown | `0` |

## Server settings

| Flag | Description | Default |
| --- | --- | --- |
| `--server.http-listen-port` | HTTP server listen port | `80` |
| `--server.grpc-listen-port` | gRPC server listen port | `9095` |

## Memberlist settings

| Flag | Description | Default |
| --- | --- | --- |
| `--memberlist.host-port` | Host port to connect to memberlist cluster | |
| `--memberlist.bind-port` | Port for memberlist to communicate on | `7946` |
| `--memberlist.message-history-buffer-bytes` | Size in bytes for the message history buffer | `0` |

## Module configuration

You can use additional flags to configuring individual Tempo modules, such as the distributor, ingester, querier, and their components. These flags follow a pattern like `--<module>.<setting>` and are extensively documented in the configuration file format.

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

