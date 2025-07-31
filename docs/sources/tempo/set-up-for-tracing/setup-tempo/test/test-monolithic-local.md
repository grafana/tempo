---
title: Validate your local Tempo deployment
menuTitle: Validate your local deployment
description: Instructions for validating your local Tempo deployment.
weight: 500
---

# Validate your local Tempo deployment

Once you've set up Grafana Tempo, the next step is to test your deployment to ensure that traces are emitted and collected correctly.
This procedure uses a Docker Compose example in the Tempo repository.

## Verify your cluster is working

To verify that Tempo is working, run the following command:

```bash
systemctl is-active tempo
```

You should see the status `active` returned. If you don't, check that the configuration file is correct, and then restart the service.
You can also use `journalctl -u tempo` to view the logs for Tempo to determine if there are any obvious reasons for failure to start.

Verify that your storage bucket has received data by signing in to your storage provider and determining that a file has been written to storage.
It should be called `tempo_cluster_seed.json`.

## Test your installation

Once Tempo is running, you can use the K6 with Traces Docker example to verify that trace data is sent to Tempo.
This procedure sets up a sample data source in Grafana to read from Tempo.

### Backend storage configuration

The Tempo examples running with docker-compose all include a version of Tempo and a storage backend like S3 and GCS.
Because Tempo is installed with a backend storage configured, you need to change the `docker-compose.yaml` file to remove Tempo and instead point trace storage to the installed version.
These steps are included in this section.

### Network configuration

Docker compose uses an internal networking bridge to connect all of the defined services. Because the Tempo instance is running as a service on the local machine host, you need the resolvable IP address of the local machine so the docker containers can use the Tempo service. You can find the host IP address of your Linux machine using a command such as `ip addr show`.

### Steps

1. Clone the Tempo repository:

   ```
   git clone https://github.com/grafana/tempo.git
   ```

1. Go into the examples directory:

   ```
   cd tempo/example/docker-compose/local
   ```

1. Edit the file `docker-compose.yaml`, and remove the `tempo` service and all its properties, so that the first service defined is `k6-tracing`. The start of your `docker-compose.yaml` should look like this:

   ```
   version: "3"
   services:

   k6-tracing:
   ```

1. Edit the `k6-tracing` service, and change the value of `ENDPOINT` to the local IP address of the machine running Tempo and docker compose, eg. `10.128.0.104:4317`. This is the OTLP gRPC port:

   ```
   environment:
     - ENDPOINT=10.128.0.104:4317
   ```

   This ensures that the traces sent from the example application go to the locally running Tempo service on the Linux machine.

1. Edit the `k6-tracing` service and remove the dependency on Tempo by deleting the following lines:

   ```
   depends_on:
   tempo
   ```

   Save the `docker-compose.yaml` file and exit your editor.

1. Edit the default Grafana data source for Tempo that is included in the examples. Edit the file located at `tempo/example/shared/grafana-datasources.yaml`, and change the `url` field of the `Tempo` data source to point to the local IP address of the machine running the Tempo service instead (eg. `url: http://10.128.0.104:3200`). The Tempo data source section should resemble this:

   ```
   - name: Tempo
     type: tempo
     access: proxy
     orgId: 1
     url: http://10.128.0.104:3200
   ```

   Save the file and exit your editor.

1. Edit the Prometheus configuration file so it uses the Tempo service as a scrape target. Change the target to the local Linux host IP address. Edit the `tempo/example/shared/prometheus.yaml` file, and alter the `tempo` job to replace `tempo:3200` with the Linux machine host IP address.

   ```yaml
     - job_name: 'tempo'
   	static_configs:
     	- targets: [ '10.128.0.104:3200' ]
   ```

   Save the file and exit your editor.\*\*

1. Start the three services that are defined in the docker-compose file:

   ```bash
   docker compose up -d
   ```

1. Verify that the services are running using `docker compose ps`. You should see something like:

   ```
   NAME             	IMAGE                                   	COMMAND              	SERVICE         	CREATED         	STATUS          	PORTS
   local-grafana-1  	grafana/grafana:9.3.2                   	"/run.sh"            	grafana         	2 minutes ago   	Up 3 seconds    	0.0.0.0:3000->3000/tcp, :::3000->3000/tcp
   local-k6-tracing-1   ghcr.io/grafana/xk6-client-tracing:v0.0.2   "/k6-tracing run /ex…"   k6-tracing      	2 minutes ago   	Up 2 seconds
   local-prometheus-1   prom/prometheus:latest                  	"/bin/prometheus --c…"   prometheus      	2 minutes ago   	Up 2 seconds    	0.0.0.0:9090->9090/tcp, :::9090->9090/tcp
   ```

   Grafana is running on port 3000, Prometheus is running on port 9090. Both should be bound to the host machine.

1. As part of the docker compose manifest, Grafana is now running on your Linux machine, reachable on port 3000. Point your web browser to the Linux machine on port 3000. You might need to port forward the local port if you’re doing this remotely, for example, via SSH forwarding.

1. Once logged in, navigate to the **Explore** page, select the Tempo data source and select the **Search** tab. Select **Run query** to list the recent traces stored in Tempo. Select one to view the trace diagram:
   {{< figure align="center" src="/media/docs/grafana/data-sources/tempo/query-editor/tempo-ds-builder-span-details-v11.png" alt="Use the query builder to explore tracing data in Grafana" >}}

1. Alter the Tempo configuration to point to the instance of Prometheus running in docker compose. To do so, edit the configuration at `/etc/tempo/config.yaml` and change the `storage` block under the `metrics_generator` section so that the remote write URL is `http://localhost:9090`. The configuration section should look like this:

   ```yaml
    storage:
        path: /var/tempo/generator/wal
        remote_write:
           - url: http://localhost:9090/api/v1/write
           send_exemplars: true

   ```

   Save the file and exit the editor.

1. Finally, restart the Tempo service by running:

   ```
   sudo systemctl restart tempo
   ```

1. A couple of minutes after Tempo has successfully restarted, select the **Service graph** tab for the Tempo data source in the **Explore** page. Select **Run query** to view a service graph, generated by Tempo’s metrics-generator.
   {{< figure align="center" src="/media/docs/grafana/data-sources/tempo/query-editor/tempo-ds-query-service-graph.png" alt="Service graph sample" >}}
