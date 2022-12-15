---
title: Deploy on Linux
menuTitle: Deploy on Linux
description: Learn how to deploy Tempo on Linux
weight: 100
---

# Deploy on Linux

This guide provides a step-by-step process for installing Tempo on Linux.
It assumes you have access to a Linux system and the permissions required to deploy a service with network and file system access.
At the end of this guide, you will have deployed a single Tempo instance on a single node.

## Before you begin

To follow this guide, you need:

- A running Grafana instance (see [installation instructions](https://grafana.com/docs/grafana/latest/setup-grafana/installation/))
- An Amazon S3 compatible object store
<!-- - Git and Docker installed to run the TNS app -->

### System requirements

This configuration is an example you can use as a starting point.
You may need to have more resources for your system than the minimum specifications listed below.
Additional adjustments will be necessary for a production environment.

You must have the permissions required to deploy a service with a network and file system access.

Your Linux system should have at least:

- 4 CPUs
- 16 GB of memory

## Set up an object storage bucket

Tempo uses object storage as the backend for its trace storage.
It also uses object storage for storing various data related to the state of the system.

Tempo supports using the local filesystem as the backend for trace storage as well.
This is not recommended for production deployments. This guide focuses on setup with an object storage.

This example uses [Amazon S3](https://docs.aws.amazon.com/AmazonS3/latest/userguide/Welcome.html) on the AWS `us-east-1` region as your object store.
If you plan on using a different region or object storage service, update the storage fields in the configuration file below. Currently, the supported object storage backends are AWS S3, other S3-compliant object stores, and Google Cloud’s GCS.

After you have provisioned an object storage backend, create the bucket `grafana-traces-data`.
The buckets will be referenced in the configuration file of this guide.
You may need to alter the bucket name to be globally unique.

Consider adding a prefix for your organization to the bucket, for example, `myorg-grafana-traces-data`, and then replacing the names in the rest of these instructions with those bucket names.

## Install Tempo

For a linux-amd64 installation, run the following commands via the command line interface on your Linux machine.
You need administrator privileges to do this by running as the `root` user or via `sudo` as a user with permissions to do so.

1. Download the tempo binary, verify checksums, and add network capabilities to the binary. Be sure to [download the correct package installation](https://github.com/grafana/tempo/releases/tag/v1.5.0) for your OS and architecture:

   ```bash
   curl -Lo tempo_1.5.0_linux_amd64.deb https://github.com/grafana/tempo/releases/download/v1.5.0/tempo_1.5.0_linux_amd64.deb
   echo 967b06434252766e424eef997162ef89257fdb232c032369ad2e644920337a8c \
     tempo_1.5.0_linux_amd64.deb | sha256sum -c
   dpkg -i tempo_1.5.0_linux_amd64.deb
   ```

## Create a Tempo configuration file

Copy the following YAML configuration to a file called `tempo.yaml`.

Paste in your S3 credentials for admin_client and the storage backend. If you wish to give your cluster a unique name, add a cluster property with the appropriate name.

Refer to the [Tempo configuration documentation]({{< relref "../configuration" >}}) for explanations of the available options.

In the following configuration, Tempo options are altered to only listen to the OTLP gRPC and HTTP protocols.
By default, Tempo listens for all compatible protocols.
The [extended instructions for installing the TNS application]({{< relref "../linux" >}}) and Grafana Agent to verify that Tempo is receiving traces, relies on the default Jaeger port being available. If Tempo were also attempting to listen on the same port as the Grafana Agent for Jaeger, then Tempo would not start due a port conflict, hence we disable listening on that port in Tempo for a single Linux node.

```yaml
   metrics_generator_enabled: true

   server:
   http_listen_port: 3200

   distributor:
   receivers:                           # this configuration will listen on all ports and protocols that tempo is capable of.
      otlp:
         protocols:
         http:
         grpc:

   ingester:
   trace_idle_period: 10s               # the length of time after a trace has not received spans to consider it complete and flush it
   max_block_bytes: 1_000_000           # cut the head block when it hits this size or ...
   max_block_duration: 5m               #   this much time passes

   compactor:
   compaction:
      compaction_window: 1h              # blocks in this time window will be compacted together
      max_block_bytes: 100_000_000       # maximum size of compacted blocks
      block_retention: 1h
      compacted_block_retention: 10m

   metrics_generator:
   registry:
      external_labels:
         source: tempo
         cluster: linux-microservices
   storage:
      path: /tmp/tempo/generator/wal
      remote_write:
         - url: http://prometheus:9090/api/v1/write
         send_exemplars: true

   storage:
   trace:
      backend: s3
      s3:
         endpoint: s3.us-east-1.amazonaws.com
         bucket: grafana-traces-data
         forcepathstyle: true
         #set to true if endpoint is https
         insecure: true
         access_key: # TODO - Add S3 access key
         secret_key: # TODO - Add S3 secret key
      block:
         bloom_filter_false_positive: .05 # bloom filter false positive rate.  lower values create larger filters but fewer false positives
         index_downsample_bytes: 1000     # number of bytes per index record
         encoding: zstd                   # block encoding/compression.  options: none, gzip, lz4-64k, lz4-256k, lz4-1M, lz4, snappy, zstd, s2
      wal:
         path: /tmp/tempo/wal             # where to store the the wal locally
         encoding: snappy                 # wal encoding/compression.  options: none, gzip, lz4-64k, lz4-256k, lz4-1M, lz4, snappy, zstd, s2
      local:
         path: /tmp/tempo/blocks
      pool:
         max_workers: 100                 # worker pool determines the number of parallel requests to the object store backend
         queue_depth: 10000

   overrides:
   metrics_generator_processors: [service-graphs, span-metrics]
```
>**Note:** that in the above configuration we enable the metrics generator to generate Prometheus metrics data from incoming trace spans. This is sent to a Prometheus remote write compatible metrics store at `http://prometheus:9090/api/v1/write` (in the `metrics_generator` configuration block). Ensure you change the relevant `url` parameter to your own Prometheus compatible storage instance, or disable the metrics generator by replacing `metrics_generator_enabled: true` with `metrics_generator_enabled: false` if you do not wish to generate span metrics.

## Move the configuration file to the proper directory

Copy the `tempo.yaml` to `/etc/tempo/config.yml`:

```bash
cp tempo.yaml /etc/tempo/config.yml
```

## Restart the tempo service

Use `systemctl` to restart the service (depending on how you installed Tempo, this may be different):

```bash
systemctl start tempo.service
```

You can replace `restart` with `stop` to stop the service, and `start` to start the service again after it's stopped, if required.

## Verify your cluster is working

To verify that Tempo is working, run the following command:

```bash
systemctl is-active tempo
```

You should see the status `active` returned. If you do not, check that the configuration file is correct, and then restart the service. You can also use `journalctl -u tempo` to view the logs for Tempo to determine if there are any obvious reasons for failure to start.

## Testing your installation

Verify that your storage bucket has received data by signing in to your storage provider and determining that a file has been written to storage. It should be called `tempo_cluster_seed.json`.

<!-- You can also [set up a test app]({{< relref "set-up-test-app">}}) to test that traces are received and visualized. -->


<!-- Need info here --=>


<!-- Does not apply to Tempo
Refer to [Set up the Tempo plugin for Grafana]({{< relref "../setup-get-plugin-grafana" >}}) to integrate your Tempo cluster with Grafana and a UI to interact with the Admin API.
-->

<!-- This section is commented out until some issues with the TNS install (dealing with ports) are addressed. >
## Test your configuration using the TNS application

You can use The New Stack (TNS) application to test GET data.
You need both git and Docker installed on your local machine.
Refer to the [Install Docker Engine](https://docs.docker.com/engine/install/) and [Installing Git](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git) documentation to do this.

The docker-compose file for the TNS contains multiple Grafana components that are not needed to test GET.
This procedure comments out unnecessary components.

To set up the TNS app:

1. Clone the repository using commands similar to the ones below:

    ```bash
      git clone git+ssh://github.com/grafana/tns
    ```

1. In the `docker-compose.yaml` manifest, alter each instance of `JAEGER_ENDPOINT` to the Grafana Agent running locally on port 14268 (the Jaeger listening port).

   ```yaml
	   JAEGER_ENDPOINT: ‘http://localhost:14268’
   ```

1. Install the Loki Docker driver plugin.

  ```bash
    docker plugin install grafana/loki-docker-driver:latest --alias loki --grant-all-permissions
   ```

1. Deploy the TNS application. We’re only starting particular components as we only want to run the TNS application instead of all of the other Grafana components (that will clash with the components we’ve already installed, including Tempo).

   ```bash
	   docker compose up loadgen app db
   ```

1. Once the application is running, look at the logs for one of the services (such as the App service) and find a relevant trace ID. For example:

   ```bash
   ~/tns/tns/production/docker-compose$ docker compose logs app
   docker-compose-app-1  | level=info http=[::]:80 grpc=[::]:9095 msg="server listening on addresses"
   docker-compose-app-1  | level=info database(s)=1
   docker-compose-app-1  | level=info msg="HTTP client success" status=200 url=http://db duration=5.496108ms traceID=28a21cef4eda3de9
   docker-compose-app-1  | level=debug traceID=28a21cef4eda3de9 msg="GET / (200) 6.144544ms"
   docker-compose-app-1  | level=info msg="HTTP client success" status=200 url=http://db duration=2.399171ms traceID=72cf668b098c8c55
   docker-compose-app-1  | level=debug traceID=72cf668b098c8c55 msg="GET / (200) 2.698249ms"
   docker-compose-app-1  | level=info msg="HTTP client success" status=200 url=http://db duration=1.708462ms traceID=628e8a4418b81409
   docker-compose-app-1  | level=debug traceID=628e8a4418b81409 msg="GET / (200) 2.163996ms"
   ```

1. Go to Grafana and select the **Explore** menu item.
1. Select the **Tempo data source** from the list of data sources.
1. Copy the trace ID into the **Trace ID** edit field.
1. Select **Run query**.
1. The trace will be displayed in the traces **Explore** panel.
-->