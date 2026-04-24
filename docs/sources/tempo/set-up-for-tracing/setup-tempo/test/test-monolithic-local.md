---
title: Validate your local Tempo deployment
menuTitle: Validate your local deployment
description: Validate your local Grafana Tempo deployment by sending traces and verifying they are stored and queryable.
weight: 500
---

# Validate your local Tempo deployment

After you've set up Grafana Tempo, the next step is to verify that traces are ingested, stored, and queryable.

This page covers two levels of validation:

- **Quick validation** uses `telemetrygen` and the Tempo HTTP API to confirm traces flow end-to-end from the command line, with no additional services required.
- **Grafana validation** connects Grafana to your local Tempo instance so you can explore traces visually.

## Before you begin

To follow this procedure, you need:

- A locally installed and running Tempo service (refer to [Deploy on Linux](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/deploy/locally/linux/))
- [OpenTelemetry `telemetrygen`](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/cmd/telemetrygen) installed for generating test traces
- A running Grafana instance (refer to [the installation instructions](/docs/grafana/<GRAFANA_VERSION>/setup-grafana/installation/)), only required for the [Grafana validation](#verify-traces-in-grafana) section

## Verify Tempo is running

1. Check the service status:

   ```bash
   systemctl is-active tempo
   ```

   You should see the status `active` returned. If you don't, check that the configuration file is correct, and then restart the service.
   You can also use `journalctl -u tempo` to view the logs for Tempo to determine if there are any obvious reasons for failure to start.

1. Verify that Tempo created the storage subdirectories:

   ```bash
   ls /data/tempo/
   ```

   You should see `wal` and `blocks` directories.

   If you configured Tempo with an S3-compatible backend instead of local storage, refer to [Verify data in your S3-compatible store](/docs/tempo/<TEMPO_VERSION>/configuration/hosted-storage/s3/#verify-data-in-your-s3-compatible-store) for storage verification steps.

## Send test traces

Use [OpenTelemetry `telemetrygen`](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/cmd/telemetrygen) to send traces to Tempo. The following command sends 100 traces (20 per second for 5 seconds) over OTLP gRPC:

```bash
telemetrygen traces --otlp-insecure --rate 20 --duration 5s --otlp-endpoint localhost:4317
```

## Verify traces using the Tempo API

After sending traces, you can verify them directly using the Tempo HTTP API. No additional services are required.

1. Search for recent traces:

   ```bash
   curl -s http://localhost:3200/api/search | jq .
   ```

   You should see a `traces` array containing your test traces. Each entry includes a `traceID`, `rootServiceName`, and `rootTraceName`. For example:

   ```json
   {
     "traces": [
       {
         "traceID": "abc123...",
         "rootServiceName": "telemetrygen",
         "rootTraceName": "lets-go",
         "startTimeUnixNano": "1776912138880042305"
       }
     ]
   }
   ```

   {{< admonition type="note" >}}
   Trace data appears in search results after the live store flushes completed blocks to disk, which can take 15–30 seconds. If the `traces` array is empty, wait and retry.
   {{< /admonition >}}

1. Retrieve a specific trace by ID using a `traceID` value from the search results:

   ```bash
   curl -s http://localhost:3200/api/v2/traces/<TRACE_ID> | jq .
   ```

   Replace `<TRACE_ID>` with an actual trace ID from the previous step.

If both commands return trace data, your Tempo installation is ingesting, storing, and serving traces correctly.

Refer to the [Tempo HTTP API documentation](/docs/tempo/<TEMPO_VERSION>/api_docs/) for the full API reference and [Push spans with HTTP](/docs/tempo/<TEMPO_VERSION>/api_docs/pushing-spans-with-http/) for an alternative approach using `curl` to send traces.

## Verify traces in Grafana

If you have a Grafana instance running, you can verify that traces are visible in the Grafana Explore view. If Grafana is not already installed, refer to [the installation instructions](/docs/grafana/<GRAFANA_VERSION>/setup-grafana/installation/). For the full Tempo data source configuration reference, refer to [Configure the Tempo data source](/docs/grafana/<GRAFANA_VERSION>/datasources/tempo/configure-tempo-data-source/).

1. In Grafana, navigate to **Connections** > **Data sources** and select **Add data source**.

1. Select **Tempo** and set the URL to `http://localhost:3200`.

1. Select **Save & Test**. You should see a message that says `Data source is working`.

1. Navigate to the **Explore** page, select the **Tempo** data source, and choose the **Search** query type.

1. Select **Run query** to list the recent traces stored in Tempo. You should see traces from `telemetrygen` in the results. Select a trace to view its spans:

   {{< figure align="center" src="/media/docs/grafana/data-sources/tempo/query-editor/tempo-ds-builder-span-details-v11.png" alt="Use the query builder to explore tracing data in Grafana" >}}

###  Optional: Metrics-generator

If you are using the metrics-generator, you can enable it to generate metrics from the traces.

1. Edit the configuration at `/etc/tempo/config.yml` and add or update the `metrics_generator` section. Set the `remote_write` URL to the address of your Prometheus-compatible storage instance. The configuration section should look like this:

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
