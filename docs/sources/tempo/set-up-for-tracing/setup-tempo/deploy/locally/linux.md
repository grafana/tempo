---
title: Deploy on Linux
menuTitle: Deploy on Linux
description: Learn how to install and configure a single Grafana Tempo instance on Linux in monolithic mode with local storage.
weight: 400
aliases:
  - ../../../../setup/linux/ # /docs/tempo/next/setup/linux/
---

# Deploy on Linux

This guide provides a step-by-step process for installing Grafana Tempo on Linux.
It assumes you have access to a Linux system and the permissions required to deploy a service with network and file system access.
At the end of this guide, you have a single Tempo instance deployed on a single node.

This procedure provides a test installation suitable for local development and evaluation.
If you plan to use this as a starting point for a production deployment, review the configuration against your organization's best practices for security, storage, retention, and availability.
If you're upgrading from Tempo 2.x, refer to [Upgrade your Tempo installation](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/upgrade/) instead.

These instructions focus on a [monolithic (single-binary) installation](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/deployment-modes/).
In single-binary mode, Tempo runs all components in one process and doesn't require Kafka.
Traces are ingested directly in-process and flushed to storage.

## Before you begin

To follow this guide, you need:

- [OpenTelemetry `telemetrygen`](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/cmd/telemetrygen) installed to [test your deployment](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/test/test-monolithic-local/)
- A running Grafana instance (refer to [the installation instructions](/docs/grafana/<GRAFANA_VERSION>/setup-grafana/installation/)) if you want to explore traces visually

### System requirements

These values are a starting point for a monolithic Tempo deployment on a single node.
They are not hard minimums for every environment, and they are not production sizing recommendations.

For the Tempo host itself, start with:

- 4 CPUs
- 4–8 GB of memory

Use 16 GB of memory or more if any of the following apply:

- You run additional local components on the same machine, such as Grafana, an object store, or Prometheus
- You enable metrics-generator
- You test moderate or high ingest rates
- You increase live-trace buffering or run heavier query workloads
- You want extra headroom for benchmarking or troubleshooting

Co-locating Tempo with Grafana, Prometheus, or other services on the same machine is fine for evaluation, but increases memory pressure. If memory is constrained, run Tempo on its own host.

Production sizing depends on your workload and infrastructure, including ingest rate, tenant count, query concurrency, retention, metrics-generator settings, and object store performance.

Validate sizing with your own load before using it in production.

## Set up storage

This guide uses the local filesystem as the storage backend. The configuration stores the write-ahead log (WAL) at `/data/tempo/wal` and trace blocks at `/data/tempo/blocks`. Tempo also uses `/var/tempo` at runtime for the live store and internal caches.

1. Create the data directories:

   ```bash
   sudo mkdir -p /data/tempo /var/tempo
   ```

1. Set the directory owner to the `tempo` user (created by the deb package):

   ```bash
   sudo chown -R tempo /data/tempo /var/tempo
   ```

Local storage is suitable for single-node evaluation and development. For production environments, use an object storage backend such as [AWS S3](/docs/tempo/<TEMPO_VERSION>/configuration/hosted-storage/s3/), [Azure Blob Storage](/docs/tempo/<TEMPO_VERSION>/configuration/hosted-storage/azure/), or [Google Cloud Storage](/docs/tempo/<TEMPO_VERSION>/configuration/hosted-storage/gcs/).

If you prefer to use an S3-compatible object store for local testing, refer to [S3-compatible local stores for testing](/docs/tempo/<TEMPO_VERSION>/configuration/hosted-storage/s3/#s3-compatible-local-stores-for-testing) for setup instructions using MinIO, SeaweedFS, or `rclone`.

## Install Tempo

For a linux-amd64 installation, run the following commands using the command line interface on your Linux machine.
You need administrator privileges to do this by running as the `root` user or via `sudo` as a user with permissions to do so.

Be sure to [download the correct package](https://github.com/grafana/tempo/releases/) for your OS and architecture. Replace `<TEMPO_VERSION_NUMBER>` with the version you want to install, for example `3.0.0`.

1. Download the Tempo binary. The following example downloads Tempo for the AMD64 (x86_64) processor architecture on a Linux distribution supporting deb packages:

   ```bash
   curl -Lo tempo_<TEMPO_VERSION_NUMBER>_linux_amd64.deb \
     https://github.com/grafana/tempo/releases/download/v<TEMPO_VERSION_NUMBER>/tempo_<TEMPO_VERSION_NUMBER>_linux_amd64.deb
   ```

1. Install the package:

   ```bash
   sudo dpkg -i tempo_<TEMPO_VERSION_NUMBER>_linux_amd64.deb
   ```

1. Optional: Verify the download against the `SHA256SUMS` file published on the [releases page](https://github.com/grafana/tempo/releases/).

## Create a Tempo configuration file

In the following configuration, Tempo is configured to listen on the OTLP gRPC and HTTP protocols.
By default, the OpenTelemetry Collector receiver binds to `localhost` instead of `0.0.0.0`.
This example binds to all interfaces. This can be a security risk if your Tempo instance is exposed to the public internet.

Refer to the [Tempo configuration documentation](/docs/tempo/<TEMPO_VERSION>/configuration/) for explanations of the available options.

{{< admonition type="tip" >}}
Tempo's configuration parser is strict about YAML indentation, especially for nested blocks like `storage.trace.wal.path`. If Tempo fails to start, check your indentation first.
{{< /admonition >}}

1. Copy the following YAML configuration to a file called `tempo.yaml`:

   ```yaml
   stream_over_http_enabled: true

   server:
     http_listen_port: 3200

   distributor:
     receivers:
       otlp:
         protocols:
           grpc:
             endpoint: "0.0.0.0:4317"
           http:
             endpoint: "0.0.0.0:4318"

   storage:
     trace:
       backend: local
       wal:
         path: /data/tempo/wal
       local:
         path: /data/tempo/blocks

   usage_report:
     reporting_enabled: false
   ```

1. Copy `tempo.yaml` to the Tempo configuration directory:

   ```bash
   sudo cp tempo.yaml /etc/tempo/config.yml
   ```

### Configuration file notes

This configuration is for monolithic mode (`-target=all`), where all required components run in one process without Kafka. Configuration blocks such as `ingest`, `block_builder`, `live_store_client`, and `backend_scheduler_client` do not apply to monolithic mode. Don't copy them from microservices examples. Refer to the [Components by deployment mode](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/deployment-modes/#components-by-deployment-mode) table for the full mapping of components and configuration blocks to each mode.

The following options are common additions to this basic configuration:

#### Block retention

With local storage, trace blocks accumulate on disk until they exceed the configured retention period. The default is 14 days (336h). If disk space is limited, set a shorter retention using the `block_retention` setting in the [compaction configuration](/docs/tempo/<TEMPO_VERSION>/configuration/#compaction).

#### Metrics-generator

The metrics-generator produces RED metrics (rate, errors, duration) and service graphs from incoming trace spans. It requires a Prometheus-compatible remote write target and enabling processors in the `overrides` block. Refer to the [metrics-generator configuration](/docs/tempo/<TEMPO_VERSION>/configuration/#metrics-generator) and the [metrics-generator documentation](/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/metrics-generator/) for setup details.

#### Ingestion limits

Tempo enforces default ingestion limits that may not fit every workload. If you see `RATE_LIMITED`, `TRACE_TOO_LARGE`, or `LIVE_TRACES_EXCEEDED` errors, you can tune these limits globally or per-tenant in the [overrides configuration](/docs/tempo/<TEMPO_VERSION>/configuration/#overrides). Refer to [Manage trace ingestion](/docs/tempo/<TEMPO_VERSION>/operations/manage-trace-ingestion/) for sizing guidance.

#### Backend worker

The `backend_worker.backend_scheduler_addr` setting is omitted from this configuration. In monolithic mode, Tempo auto-configures the backend worker to connect to the scheduler on the native gRPC port (default `9095`). Setting it explicitly to the HTTP port can produce noisy polling logs.

## Start the Tempo service

These `systemctl` instructions apply to monolithic mode only. Running individual Tempo components as separate systemd services (microservices mode) is not covered here.

1. Use `systemctl` to restart the service (depending on how you installed Tempo, this may be different):

   ```bash
   sudo systemctl restart tempo.service
   ```

   You can replace `restart` with `stop` to stop the service, and `start` to start the service again after it's stopped, if required.

1. Verify that Tempo is running:

   ```bash
   systemctl is-active tempo
   ```

   You should see the status `active` returned. If you don't, check that the configuration file is correct, and then restart the service.
   You can also use `journalctl -u tempo` to view the logs for Tempo to determine if there are any obvious reasons for failure to start.

1. Verify that Tempo created the storage subdirectories:

   ```bash
   ls /data/tempo/
   ```

   You should see `wal` and `blocks` directories. Trace data appears in `blocks` after you send traces and the live store flushes them to disk, which can take 15–30 seconds.

## Test your installation

To verify that traces flow through Tempo correctly, refer to [Validate your local Tempo deployment](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/test/test-monolithic-local/).

## Next steps

After you validate your Tempo deployment, consider exploring these topics:

- [Instrument for distributed tracing](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/instrument-send/) to send traces from your own applications
- [Configure Tempo](/docs/tempo/<TEMPO_VERSION>/configuration/) to customize settings for your environment
- [Set up monitoring for Tempo](/docs/tempo/<TEMPO_VERSION>/operations/monitor/set-up-monitoring/) to observe your Tempo instance with dashboards and alerts
