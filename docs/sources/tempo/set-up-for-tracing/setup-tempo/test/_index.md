---
title: Validate deployment
menuTitle: Validate deployment
description: Instructions for validating your Tempo deployment
weight: 500
---

<!-- This page is not finished. It's hidden from the published doc site by draft: true. -->

# Validate your Tempo deployment

To test your Tempo deployment, select one of the procedures below to test your Tempo deployment:

{{< section withDescriptions="true">}}

<!-- Update these steps before publishing. They aren't complete.
Follow these steps to ensure that traces are being sent and received correctly.
This guide assumes you have already set up a Tempo instance and have Grafana configured to query it.

For additional information, refer to [Push spans with HTTP](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/push-spans-with-http/).

If you are using Cloud Traces, refer to [Set up Cloud Traces](https://grafana.com/docs/grafana-cloud/send-data/traces/set-up/).

## Before you begin


## Test using the Tempo CLI
To test your Tempo deployment, you can use the Tempo CLI to push traces to your Tempo instance.
The Tempo CLI is a command-line tool that allows you to interact with your Tempo instance and perform various operations, such as pushing traces, querying traces, and more.

### Install the Tempo CLI
You can install the Tempo CLI by following the instructions in the [Tempo CLI documentation](https://grafana.com/docs/tempo/latest/operations/cli/).

### Push traces to Tempo

To push traces to your Tempo instance, you can use the `tempo push` command. For example, to push a trace with a specific trace ID and span ID, you can use the following command:
```bash
tempo push --trace-id <TRACE_ID> --span-id <SPAN_ID> --endpoint http://<TEMPO-DISTRIBUTOR-SERVICE>:<HTTP-LISTEN-PORT>
```
Replace `<TRACE_ID>` and `<SPAN_ID>` with the actual trace ID and span ID you want to push, and `<TEMPO-DISTRIBUTOR-SERVICE>` and `<HTTP-LISTEN-PORT>` with the appropriate values for your Tempo instance.


1. **Use `curl` to Query Tempo**: You can use `curl` to query the Tempo API and check if traces are being received. For example:
   ```bash
   curl -G http://<TEMPO-QUERY-FRONTEND-SERVICE>:<HTTP-LISTEN-PORT>/api/traces
   ```


## Test using Grafana

You can use Grafana to check if traces are sent and received.

### Test if traces are sent

To test if traces are being sent to Tempo, you can use the following methods:


2. **Use OpenTelemetry Collector**: If you have an OpenTelemetry Collector configured, you can check its logs to see if it is successfully exporting traces to Tempo. Ensure that the collector is configured with the correct endpoint for Tempo.

3. **Use Grafana Explore**: Navigate to the **Explore** section in Grafana and select the Tempo data source. You can run a query to see if traces are being returned.


### Test if traces are received

To test if traces are being received by Tempo, you can use the following methods:

1. **Grafana Explore**: Navigate to the **Explore** section in Grafana and select the Tempo data source. You can run a query to see if traces are being returned.
2. **Use `curl` to Query Tempo**: You can use `curl` to query the Tempo API and check if traces are being returned. For example:
   ```bash
   curl -G http://<TEMPO-QUERY-FRONTEND-SERVICE>:<HTTP-LISTEN-PORT>/api/traces
   ```
4. **Use OpenTelemetry Collector**: If you have an OpenTelemetry Collector configured, you can check its logs to see if it is successfully exporting traces to Tempo. Ensure that the collector is configured with the correct endpoint for Tempo.
5. **Check Grafana Logs**: Look for logs in the Grafana instance that indicate traces are being returned from Tempo. You can find these logs in the Grafana UI under **Configuration** > **Logs**.

-->
