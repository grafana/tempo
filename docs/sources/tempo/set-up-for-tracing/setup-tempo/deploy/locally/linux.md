---
title: Deploy on Linux
menuTitle: Deploy on Linux
description: Learn how to deploy a single Tempo instance on a single node on Linux.
weight: 400
aliases:
  - ../../../../setup/linux/ # /docs/tempo/next/setup/linux/
---

# Deploy on Linux

This guide provides a step-by-step process for installing Grafana Tempo on Linux.
It assumes you have access to a Linux system and the permissions required to deploy a service with network and file system access.
At the end of this guide, you have a single Tempo instance deployed on a single node.

These instructions focus on a [monolithic (single-binary) installation](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/deployment-modes/).
In single-binary mode, Tempo runs all components in one process and doesn't require Kafka.
Traces are ingested directly in-process and flushed to object storage.

## Before you begin

To follow this guide, you need:

- A running Grafana instance (refer to [the installation instructions](/docs/grafana/<GRAFANA_VERSION>/setup-grafana/installation/))
- An S3-compatible object store, such as [MinIO](https://min.io/), and the [MinIO Client (`mc`)](https://min.io/docs/minio/linux/reference/minio-mc.html) to create buckets

  {{< admonition type="note" >}}
  MinIO has been deprecated. These examples still use MinIO, but a replacement S3-compatible object store is under evaluation. This page will be updated when an alternative is selected.
  {{< /admonition >}}

- Git, Docker, and the docker-compose plugin installed to [test your deployment](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/test/test-monolithic-local/)

### System requirements

These values are a starting point for a monolithic Tempo deployment on a single node.
They are not hard minimums for every environment, and they are not production sizing recommendations.

For the Tempo host itself, start with:

- 4 CPUs
- 4–8 GB of memory

Use 16 GB of memory or more if any of the following apply:

- You run additional local components on the same machine, such as Grafana, MinIO, or Prometheus
- You enable metrics-generator
- You test moderate or high ingest rates
- You increase live-trace buffering or run heavier query workloads
- You want extra headroom for benchmarking or troubleshooting

Production sizing depends on your workload and infrastructure, including ingest rate, tenant count, query concurrency, retention, metrics-generator settings, and object store performance.

Validate sizing with your own load before using it in production.

## Set up an object storage bucket

Tempo uses object storage as the backend for its trace storage.
It also uses object storage for storing various data related to the state of the system.

Tempo supports using the local filesystem as the backend for trace storage as well.
This isn't recommended for production deployments. This guide focuses on setup with an object storage.

This example uses [MinIO](https://min.io/), an S3-compatible object store you can run locally.
If you plan on using a different object storage service, update the storage fields in the configuration file below. The supported object storage backends are [AWS S3](https://aws.amazon.com/), S3-compliant object stores like MinIO, [Azure](https://azure.microsoft.com), and [Google Cloud Storage](https://cloud.google.com/).

To set up MinIO for this example:

1. Install MinIO by following the [MinIO quickstart guide](https://min.io/docs/minio/linux/index.html).

1. Create a data directory and start MinIO:

   ```bash
   sudo mkdir -p /data/minio
   sudo chown -R $USER:$USER /data
   minio server /data --console-address ':9001'
   ```

   By default, MinIO uses `minioadmin` for both the access key and secret key. MinIO runs in the foreground, so open a new terminal for the remaining steps.

1. Create a bucket called `tempo` using the MinIO Client (`mc`):

   ```bash
   mc alias set local http://localhost:9000 minioadmin minioadmin
   mc mb local/tempo
   ```

## Install Tempo

For a linux-amd64 installation, run the following commands via the command line interface on your Linux machine.
You need administrator privileges to do this by running as the `root` user or via `sudo` as a user with permissions to do so.

Download the Tempo binary and install it. Be sure to [download the correct package](https://github.com/grafana/tempo/releases/) for your OS and architecture. Replace `<TEMPO_VERSION_NUMBER>` with the version you want to install, for example `3.0.0`.

The following example downloads and installs Tempo for the AMD64 (x86_64) processor architecture on a Linux distribution supporting deb packages:

   ```bash
   curl -Lo tempo_<TEMPO_VERSION_NUMBER>_linux_amd64.deb \
     https://github.com/grafana/tempo/releases/download/v<TEMPO_VERSION_NUMBER>/tempo_<TEMPO_VERSION_NUMBER>_linux_amd64.deb
   sudo dpkg -i tempo_<TEMPO_VERSION_NUMBER>_linux_amd64.deb
   ```

You can verify the download against the `SHA256SUMS` file published on the [releases page](https://github.com/grafana/tempo/releases/).

## Create a Tempo configuration file

Copy the following YAML configuration to a file called `tempo.yaml`.

Refer to the [Tempo configuration documentation](/docs/tempo/<TEMPO_VERSION>/configuration/) for explanations of the available options.

In the following configuration, Tempo is configured to listen on the OTLP gRPC and HTTP protocols.
By default, the OpenTelemetry Collector receiver binds to `localhost` instead of `0.0.0.0`.
This example binds to all interfaces. This can be a security risk if your Tempo instance is exposed to the public internet.

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

backend_scheduler:
  provider:
    compaction:
      compaction:
        block_retention: 1h

backend_worker:
  backend_scheduler_addr: localhost:3200
  compaction:
    block_retention: 1h

querier:
  query_live_store: true

metrics_generator:
  registry:
    external_labels:
      source: tempo
      cluster: linux-monolithic
  storage:
    path: /tmp/tempo/generator/wal
    remote_write:
      - url: http://<PROMETHEUS_URL>/api/v1/write
        send_exemplars: true

storage:
  trace:
    backend: s3
    s3:
      endpoint: localhost:9000
      bucket: tempo
      access_key: minioadmin
      secret_key: minioadmin
      insecure: true
    wal:
      path: /var/tempo/wal

overrides:
  defaults:
    metrics_generator:
      processors: [service-graphs, span-metrics]

usage_report:
  reporting_enabled: false
```

{{< admonition type="note" >}}
The metrics-generator is enabled in this configuration to generate Prometheus metrics data from incoming trace spans. Replace `<PROMETHEUS_URL>` with the address of your Prometheus-compatible storage instance (for example, `localhost:9090`).
To disable the metrics-generator, remove the `processors` list from the overrides and the `metrics_generator` block.
{{< /admonition >}}

## Move the configuration file to the proper directory

Copy the `tempo.yaml` to `/etc/tempo/config.yml`:

```bash
sudo cp tempo.yaml /etc/tempo/config.yml
```

## Restart the Tempo service

Use `systemctl` to restart the service (depending on how you installed Tempo, this may be different):

```bash
sudo systemctl restart tempo.service
```

You can replace `restart` with `stop` to stop the service, and `start` to start the service again after it's stopped, if required.

## Verify your cluster is working

To verify that Tempo is working, run the following command:

```bash
systemctl is-active tempo
```

You should see the status `active` returned. If you don't, check that the configuration file is correct, and then restart the service.
You can also use `journalctl -u tempo` to view the logs for Tempo to determine if there are any obvious reasons for failure to start.

After traces start flowing, verify that your storage bucket has received data. Open the MinIO Console at `http://localhost:9001` and check the `tempo` bucket for files such as `work.json` and a tenant data directory.

## Next steps

To validate that your Tempo deployment is working correctly, refer to [Validate your local Tempo deployment](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/test/test-monolithic-local/).
