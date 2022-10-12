---
title: Deploy on Linux
menuTitle: Deploy on Linux
description: Learn how to deploy GET on Linux
weight: 100
---

# Deploy on Linux

This guide provides a step-by-step process for installing Tempo on Linux.
It assumes you have access to a Linux machine and the permissions required to deploy a service with network and filesystem access.
At the end of this guide, you will have deployed a single Tempo instance on a single node.

## Before you begin

To follow this guide, you need:

- A running Grafana instance (see [installation instructions](https://grafana.com/docs/grafana/latest/setup-grafana/installation/))
- An Amazon S3 compatible object store
<!-- - Git and Docker installed to run the TNS app -->

### System prerequisites

This configuration is an example you can use as a starting point.
You may need to have more resources for your system than the minimum specifications listed below.
Additional adjustments will be necessary for a production environment.

You must have the permissions required to deploy a service with a network and file system access.

Your Linux system should have at least:

- 4 CPUs
- 16 GB of memory

## Setup an object storage bucket

Tempo uses object storage as the backend for its trace storage.
It also uses object storage for storing various administrative credentials and data related to the state of the system.

Tempo support using the local filesystem as the backend for trace storage as well.
This is not recommended for production deployments and is not supported for storing admin credentials, this guide focuses on setup with an object storage.

This example uses [Amazon S3](https://docs.aws.amazon.com/AmazonS3/latest/userguide/Welcome.html) on the AWS `us-east-1` region as your object store.
If you plan on using a different region or object storage service, update the storage fields in the configuration file below. Currently, the supported object storage backends are AWS S3, other S3-compliant object stores, and Google Cloud’s GCS.

After you have provisioned an object storage backend, create two buckets: `grafana-traces-admin` and `grafana-traces-data`.
These buckets will be referenced in the configuration file of this guide.
You may need to alter the bucket names to be globally unique.

Consider adding a prefix for your organization to the bucket, for example, `myorg-grafana-traces-admin` and `myorg-grafana-traces-data`, and then replacing the names in the rest of these instructions with those bucket names.

## Install GET

For a linux-amd64 installation, run the following commands via the command line interface on your Linux machine.
You need administrator privileges to do this by running as the `root` user or via `sudo` as a user with permissions to do so.

1. Add a dedicated user and group and then change the password for the user to `enterprise-traces`:
   ```bash
     groupadd --system enterprise-traces
     useradd --system --home-dir /var/lib/enterprise-traces -g enterprise-traces enterprise-traces
     yes enterprise-traces | passwd enterprise-traces
   ```

1. Create directories and assign ownership.

   ```bash
     mkdir -p /etc/enterprise-traces /var/lib/enterprise-traces /var/lib/enterprise-traces/rules-temp /var/lib/enterprise-traces/wal/search
     chown root:enterprise-traces /etc/enterprise-traces
     chown enterprise-traces:enterprise-traces /var/lib/enterprise-traces /var/lib/enterprise-traces/rules-temp /var/lib enterprise-traces/wal /var/lib/enterprise-traces/wal/search
     chmod 0750 /etc/enterprise-traces /var/lib/enterprise-traces /var/lib/enterprise-traces/wal /var/lib/enterprise-traces/wal/search
   ```

1. Download the enterprise-traces binary, verify checksums, and add network capabilities to the binary:

   ```bash
   curl -Lo /usr/local/bin/enterprise-traces \
   https://dl.grafana.com/get/releases/enterprise-traces-v1.3.0-linux-amd64
   echo d950922d2038c84620ebe63a21786b9eaf8d4ed5f1801e6664133520407e5e86 \
     /usr/local/bin/enterprise-traces | sha256sum -c
   chmod 0755 /usr/local/bin/enterprise-traces
   setcap 'cap_net_bind_service=+ep' /usr/local/bin/enterprise-traces
   ```

1. Set up systemd unit and enable startup on boot:

   ```bash
   cat > /etc/systemd/system/enterprise-traces.service <<EOF
   [Unit]
   After=network.target

   [Service]
   User=enterprise-traces
   Group=enterprise-traces
   WorkingDirectory=/var/lib/enterprise-traces
   ExecStart=/usr/local/bin/enterprise-traces \
   -config.file=/etc/enterprise-traces/enterprise-traces.yaml \
   -log.level=warn \

   [Install]
   WantedBy=default.target
   EOF

   systemctl daemon-reload
   systemctl enable enterprise-traces.service
   ```

## Create a GET configuration file

Copy the following YAML configuration to a file called `enterprise-traces.yaml`.

Paste in your S3 credentials for admin_client and the storage backend. If you wish to give your cluster a unique name, add a cluster property with the appropriate name. If you do not add a cluster name this will be taken automatically from the license.
By default, the `cluster_name` Update the `cluster_name` field with the name of the cluster your license was issued for and paste in your S3 credentials for the `admin_client`.

Refer to the [Tempo configuration documentation]({<< relref "../../configuration" >>}) for explanations of the available options.

In the following configuration, Tempo options are altered to only listen to the OTLP gRPC and HTTP protocols.
By default, Tempo listens for all compatible protocols.
The extended instructions for installing the TNS application and Grafana Agent to verify that Tempo is receiving traces relies on the default Jaeger port being available, hence disabling listening on that port in GET for a single Linux node.

```yaml
multitenancy_enabled: true
auth:
  type: enterprise

license:
  path: /etc/enterprise-traces/license.jwt

http_api_prefix: /tempo

distributor:
  receivers:
    otlp:
      protocols:
        grpc:
        http:

ingester:
    lifecycler:
      ring:
        replication_factor: 3

server:
    http_listen_port: 3200

storage:
    trace:
        backend: s3
        s3:
          endpoint: s3.us-east-1.amazonaws.com
          bucket: grafana-traces-data
          forcepathstyle: true
          #set to true if endpoint is https
          insecure: true
          access_key: # TODO: insert your key id
          secret_key: # TODO: insert your secret key
        wal:
          path: /var/lib/enterprise-traces/wal

admin_api:
  leader_election:
    enabled: false

admin_client:
  storage:
    s3:
      endpoint: s3.us-east-1.amazonaws.com
      bucket_name: grafana-traces-admin
      access_key_id: # TODO: insert your key id
      secret_access_key: # TODO: insert your secret key
    type: s3
```

## Move the configuration file and license to the proper directory

The enterprise-traces.yaml and license.jwt files need to be moved: 

- `enterprise-traces.yaml` should be copied to `/etc/enterprise-traces/enterprise-traces.yaml`
- `license.jwt` should be copied to `/etc/enterprise-traces/license.jwt`

Copy the configuration and the license files to all nodes in the GET cluster:

```bash
cp enterprise-traces.yaml /etc/enterprise-traces/enterprise-traces.yaml
cp license.jwt /etc/enterprise-traces/license.jwt
```

## Generate an admin token

1. Generate an admin token by running the following on a single node in the cluster, using the password for the `enterprise-traces` user set earlier:

   ```bash
   su enterprise-traces -c "/usr/local/bin/enterprise-traces \
      --config.file=/etc/enterprise-traces/enterprise-traces.yaml \
      --license.path=/etc/enterprise-traces/license.jwt \
      --log.level=warn \
      --target=tokengen"
   # Token created:  YWRtaW4tcG9saWN5LWJvb3RzdHJhcC10b2tlbjo8Ujc1IzQyfXBfMjd7fDIwMDRdYVxgeXw=
   ```

1. After you enter your password, the system outputs a new token. Save this token somewhere secure for future API calls and to enable the GET plugin.

   ```bash
   Password:
   Token created: YourTokenHere12345
   ```

1. You can export the API token to use later in the procedure using the command below. Replace the value for the token with the one you generated.

   ```bash
   export API_TOKEN=YourTokenHere12345
   ```

## Start the enterprise-traces service

Use `systemctl` to start the service:

```bash
systemctl start enterprise-traces.service
```

You can replace `start` with `stop` to stop the service.

## Verify your cluster is working

To verify your cluster is working, run the following command using the token you generated in the previous step.

```bash
curl -u :$API_TOKEN localhost:3200/ready
```

After running the above command, you should see the following output within 30-60 seconds:

```bash
ready
```

This indicates the ingester component is ready to receive trace data.

## Use the CLI to send Tempo data to Grafana

Refer to [Set up the GET plugin for Grafana]({{< relref "../setup-get-plugin-grafana" >}}) to integrate your GET cluster with Grafana and a UI to interact with the Admin API.

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
1. Select the **GET data source** from the list of data sources.
1. Copy the trace ID into the **Trace ID** edit field.
1. Select **Run query**. 
1. The trace will be displayed in the traces **Explore** panel.
-->