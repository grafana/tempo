---
title: Validate your local Tempo deployment
menuTitle: Validate your local deployment
description: Validate your local Grafana Tempo deployment by sending traces and verifying data in your S3-compatible object store.
weight: 500
---

# Validate your local Tempo deployment

After you've set up Grafana Tempo, the next step is to test your deployment to ensure that your application emits traces and Tempo collects them correctly.
This procedure uses a Docker Compose example in the Tempo repository.

## Before you begin

To follow this procedure, you need:

- A locally installed and running Tempo service with an S3-compatible object store (refer to [Deploy on Linux](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/deploy/locally/linux/))

  {{< admonition type="note" >}}
  These examples use MinIO as the default S3-compatible object store. [SeaweedFS](https://github.com/seaweedfs/seaweedfs) and [rclone serve s3](https://rclone.org/commands/rclone_serve_s3/) are provided as alternatives for local testing but have not been fully tested with Tempo. They aren't recommended for production use. Refer to [Deploy on Linux](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/deploy/locally/linux/) for setup instructions for each option.
  {{< /admonition >}}

- Git, Docker, and the docker-compose plugin installed on the same machine
- The host machine's IP address, which Docker containers use to reach the locally running Tempo service. You can find it using `ip addr show`.

## Verify your cluster is working

To verify that Tempo is working, run the following command:

```bash
systemctl is-active tempo
```

You should see the status `active` returned. If you don't, check that the configuration file is correct, and then restart the service.
You can also use `journalctl -u tempo` to view the logs for Tempo to determine if there are any obvious reasons for failure to start.

After traces start flowing, verify that your storage bucket has received data:

{{< tabs >}}
{{< tab-content name="MinIO" >}}

Open the MinIO Console at `http://localhost:9001` and check the `tempo` bucket for files such as `work.json` and a tenant data directory.

{{< /tab-content >}}
{{< tab-content name="SeaweedFS" >}}

Open the SeaweedFS admin UI at `http://localhost:23646`, or list the bucket contents using the AWS CLI:

```bash
aws --endpoint-url http://localhost:8333 s3 ls s3://tempo/ --recursive --no-sign-request
```

You should see files such as `single-tenant/<block-id>/data.parquet` and `single-tenant/<block-id>/meta.json`.

{{< /tab-content >}}
{{< tab-content name="rclone serve s3 (experimental)" >}}

List the bucket contents using the AWS CLI:

```bash
AWS_ACCESS_KEY_ID=tempokey AWS_SECRET_ACCESS_KEY=temposecret \
  aws --endpoint-url http://localhost:8080 s3 ls s3://tempo/ --recursive
```

You should see files such as `single-tenant/<block-id>/data.parquet` and `single-tenant/<block-id>/meta.json`. `rclone serve s3` does not provide a web UI.

{{< /tab-content >}}
{{< /tabs >}}

## Test your installation

After Tempo is running, you can use the Docker Compose examples in the Tempo repository to verify that trace data is sent to Tempo.
This procedure runs a trace generator, Grafana, and Prometheus alongside your locally installed Tempo service.

### Network configuration

Docker Compose uses an internal networking bridge to connect all defined services. Because the Tempo instance is running as a service on the local machine host, you need the resolvable IP address of the local machine so the Docker containers can reach the Tempo service.

You can find the host IP address of your Linux machine using a command such as `ip addr show`.

### Steps

1. Clone the Tempo repository:

   ```bash
   git clone https://github.com/grafana/tempo.git
   ```

1. Go into the single-binary examples directory:

   ```bash
   cd tempo/example/docker-compose/single-binary
   ```

1. Edit the file `docker-compose.yaml` and remove the `tempo`, `minio`, `alloy`, and `vulture` services. Keep only `k6-tracing`, `prometheus`, and `grafana`. The example includes its own `minio` service for object storage. Remove it along with the other listed services, since you already have a local object store running from the [Deploy on Linux](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/deploy/locally/linux/) setup.

1. In the `k6-tracing` service, remove the `depends_on` block and change the value of `ENDPOINT` to the local IP address of the machine running Tempo, for example, `10.128.0.104:4317`. This is the OTLP gRPC port:

   ```yaml
   environment:
     - ENDPOINT=10.128.0.104:4317
   ```

   This ensures that the traces sent from the example application go to the locally running Tempo service.

1. Edit the `grafana-datasources.yaml` file and change the `url` field in both Tempo data sources (`Tempo (yes streaming)` and `Tempo (no streaming)`) to the local IP address of the machine running the Tempo service. For example, change `http://tempo:3200` to `http://10.128.0.104:3200`. Both data sources already include a `serviceMap` section that links to Prometheus for the Service Graph feature. After editing, a Tempo data source entry should resemble this:

   ```yaml
   - name: Tempo (yes streaming)
     type: tempo
     access: proxy
     orgId: 1
     url: http://10.128.0.104:3200
     jsonData:
       httpMethod: GET
       serviceMap:
         datasourceUid: prometheus
       streamingEnabled:
         search: true
         metrics: true
   ```

   Save the file and exit your editor.

1. Edit the `prometheus.yaml` file so it uses the Tempo service as a scrape target. Change the target to the local Linux host IP address:

   ```yaml
   - job_name: 'tempo'
     static_configs:
       - targets: ['10.128.0.104:3200']
   ```

   Save the file and exit your editor.

1. Start the services defined in the Docker Compose file:

   ```bash
   docker compose up -d
   ```

1. Verify that the services are running using `docker compose ps`. You should see Grafana running on port 3000 and Prometheus on port 9090, both bound to the host machine.

1. Point your web browser to the Linux machine on port 3000. You might need to port forward the local port if you're doing this remotely, for example, via SSH forwarding.

1. Navigate to the **Explore** page, select the Tempo data source, and select the **Search** tab. Select **Run query** to list the recent traces stored in Tempo. Select one to view the trace diagram:
   {{< figure align="center" src="/media/docs/grafana/data-sources/tempo/query-editor/tempo-ds-builder-span-details-v11.png" alt="Use the query builder to explore tracing data in Grafana" >}}

1. Alter the Tempo configuration to point to the instance of Prometheus running in Docker Compose. Edit the configuration at `/etc/tempo/config.yml` and change the `url` in the `remote_write` block under the `metrics_generator` section to `http://localhost:9090/api/v1/write`. The configuration section should look like this:

   ```yaml
   metrics_generator:
     storage:
       path: /tmp/tempo/generator/wal
       remote_write:
         - url: http://localhost:9090/api/v1/write
           send_exemplars: true
   ```

   Save the file and exit the editor.

1. Restart the Tempo service:

   ```bash
   sudo systemctl restart tempo
   ```

1. After a couple of minutes, select the **Service graph** tab for the Tempo data source in the **Explore** page. Select **Run query** to view a service graph, generated by the Tempo metrics-generator.
   {{< figure align="center" src="/media/docs/grafana/data-sources/tempo/query-editor/tempo-ds-query-service-graph.png" alt="Service graph sample" >}}

## Next steps

After you validate your local Tempo deployment, consider exploring these topics:

- [Instrument for distributed tracing](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/instrument-send/) to send traces from your own applications instead of a test generator
- [Configure Tempo](/docs/tempo/<TEMPO_VERSION>/configuration/) to customize settings for your environment
- [Set up monitoring for Tempo](/docs/tempo/<TEMPO_VERSION>/operations/monitor/set-up-monitoring/) to observe your Tempo instance with dashboards and alerts
- [Use tracing data in Grafana](/docs/tempo/<TEMPO_VERSION>/configuration/use-trace-data/) to learn more about querying and visualizing traces
