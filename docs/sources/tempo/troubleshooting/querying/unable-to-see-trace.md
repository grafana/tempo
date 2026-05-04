---
title: Unable to find traces
description: Troubleshoot missing traces
weight: 473
aliases:
- ../../operations/troubleshooting/missing-trace/ # https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/troubleshooting/missing-trace/
- ../../operations/troubleshooting/unable-to-see-trace/ # https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/troubleshooting/unable-to-see-trace/
- ../unable-to-see-trace/ # htt/docs/tempo/<TEMPO_VERSION>/troubleshooting/unable-to-see-trace/
---

# Unable to find traces

The two main causes of missing traces are:

- Issues in ingestion of the data into Tempo. Spans are either not sent correctly to Tempo or they aren't getting sampled.
- Issues querying for traces that have been received by Tempo.

## Section 1: Diagnose and fix ingestion issues

The first step is to check whether the application spans are actually reaching Tempo.

Add the following flag to the distributor container - [`distributor.log_received_spans.enabled`](https://github.com/grafana/tempo/blob/57da4f3fd5d2966e13a39d27dbed4342af6a857a/modules/distributor/config.go#L55).

This flag enables debug logging of all the traces received by the distributor. These logs can help check if Tempo is receiving any traces at all.

You can also check the following metrics:

- `tempo_distributor_spans_received_total`
- `tempo_live_store_traces_created_total`

The value of both metrics should be greater than `0` within a few minutes of the application spinning up.
You can check both metrics using either:

- The metrics page exposed from Tempo at `http://<tempo-address>:<tempo-http-port>/metrics` or
- In Prometheus, if it's used to scrape metrics.

### Case 1 - `tempo_distributor_spans_received_total` is 0

If the value of `tempo_distributor_spans_received_total` is 0, possible reasons are:

- Use of incorrect protocol/port combination while initializing the tracer in the application.
- Tracing records not getting picked up to send to Tempo by the internal sampler.
- Application is running inside docker and sending traces to an incorrect endpoint.

Receiver specific traffic information can also be obtained using `tempo_receiver_accepted_spans` which has a label for the receiver (protocol used for ingestion. Ex: `jaeger-thrift`).

### Solutions

There are three possible solutions: protocol or port problems, sampling issues, or incorrect endpoints.

To fix protocol or port problems:

- Find out which communication protocol is being used by the application to emit traces. This is unique to every client SDK. For instance: Jaeger Golang Client uses `Thrift Compact over UDP` by default.
- Check the list of supported protocols and their ports and ensure that the correct combination is being used.

To fix sampling issues:

- These issues can be tricky to determine because most SDKs use a probabilistic sampler by default. This may lead to just one in a 1000 records being picked up.
- Check the sampling configuration of the tracer being initialized in the application and make sure it has a high sampling rate.
- Some clients also provide metrics on the number of spans reported from the application, for example `jaeger_tracer_reporter_spans_total`. Check the value of that metric if available and make sure it's greater than zero.
- Another way to diagnose this problem would be to generate lots and lots of traces to see if some records make their way to Tempo.

To fix an incorrect endpoint issue:

- If the application is also running inside docker, make sure the application is sending traces to the correct endpoint (`tempo:<receiver-port>`).

## Case 2 - tempo_live_store_traces_created_total is 0

If the value of `tempo_live_store_traces_created_total` is 0, this can indicate issues between the distributors and Kafka, or between Kafka and the live-stores.

### Solution

- Check distributor logs for Kafka write errors such as `msg="failed to write to kafka"`.
- Verify that Kafka is healthy and that the distributors can reach it.
- Check live-store logs to ensure they are consuming from Kafka successfully. Look for consumer lag metrics to confirm data is flowing.

## Case 3 - Live-store Kafka lag

If the live-store is lagging behind its Kafka partition, queries for recent data may return incomplete results.

To check whether lag is affecting queries, run the following PromQL query in Grafana or Prometheus:

```promql
rate(tempo_live_store_lagged_requests_total[5m])
```

A non-zero rate means that query time ranges are overlapping with the live-store's Kafka lag, and some recently ingested traces may be missing from results. The metric is labeled by `route`, so you can see which query type is affected (`/tempopb.Querier/SearchRecent` for search queries or `/tempopb.Metrics/QueryRange` for TraceQL metrics queries).

### Solution

- Check the raw consumer lag per partition using your live-store consumer group label:

  ```promql
  tempo_ingest_group_partition_lag{group="<CONSUMER_GROUP>"}
  ```

  The `group` label is derived from the live-store ring instance ID. For example, in a zone-aware deployment the group might be `live-store-zone-a`.

- If lag is persistent, the live-store may need more resources or partitions may need to be redistributed.
- To make incomplete results explicit, set `fail_on_high_lag: true` in the [live-store configuration](/docs/tempo/<TEMPO_VERSION>/configuration/#live-store). When enabled, the live-store returns an error instead of silently incomplete results.

## Case 4 - Trace is not recent

Live-stores only serve recent data. Older traces are stored in blocks built by the block-builder. If a trace was ingested but can't be found, the block-builder may not be flushing blocks to the backend correctly.

### Solution

- Check block-builder logs for errors during block creation or flushing to object storage.
- Verify the block-builder is consuming from Kafka by checking consumer lag metrics.
- Check the `tempo_block_builder_flushed_blocks` metric to confirm blocks are being written to the backend.
- Check the `tempo_block_builder_fetch_errors_total` metric for Kafka fetch issues.

## Diagnose and fix sampling and limits issues

If you are able to query some traces in Tempo but not others, you have come to the right section.

This could happen because of a number of reasons and some have been detailed in this blog post:
[Where did all my spans go? A guide to diagnosing dropped spans in Jaeger distributed tracing](/blog/2020/07/09/where-did-all-my-spans-go-a-guide-to-diagnosing-dropped-spans-in-jaeger-distributed-tracing/).
This is useful if you are using the Jaeger Agent.

If you are using Grafana Alloy, continue reading the following section for metrics to monitor.

### Diagnose the issue

Check if the pipeline is dropping spans. The following metrics on Grafana Alloy help determine this:

- `exporter_send_failed_spans_ratio_total`. The value of this metric should be `0`.
- `receiver_refused_spans_ratio_total`. This value of this metric should be `0`.

If the pipeline isn't reporting any dropped spans, check whether application spans are being dropped by Tempo. The following metrics help determine this:

- `tempo_receiver_refused_spans`. The value of `tempo_receiver_refused_spans` should be `0`.

  If the value of `tempo_receiver_refused_spans` is greater than 0, then the possible reason is the application spans are being dropped due to rate limiting.

### Solutions

- If the pipeline (Grafana Alloy) drops spans, the deployment may need to be scaled up.
- There might also be issues with connectivity to Tempo backend, check Alloy logs and make sure the Tempo endpoint and credentials are correctly configured.
- If Tempo drops spans, this may be due to rate limiting.
  Rate limiting may be appropriate and therefore not an issue. The metric simply explains the cause of the missing spans.
- If you require a higher ingest volume, increase the configuration for the rate limiting by adjusting the `max_traces_per_user` property in the [configured override limits](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#standard-overrides).

{{< admonition type="note" >}}
Check the [ingestion limits page](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/#overrides) for further information on limits.
{{< /admonition >}}

## Section 3: Diagnose and fix issues with querying traces

If Tempo is correctly ingesting trace spans, then it's time to investigate possible issues with querying the data.

Check the logs of the query-frontend. The query-frontend pod runs with two containers, `query-frontend` and `query`.
Use the following command to view query-frontend logs:

```bash
kubectl logs -f pod/query-frontend-xxxxx -c query-frontend
```

The presence of the following errors in the log may explain issues with querying traces:

- `level=info ts=XXXXXXX caller=frontend.go:63 method=GET traceID=XXXXXXXXX url=/api/traces/XXXXXXXXX duration=5m41.729449877s status=500`
- `no org id`
- `could not dial 10.X.X.X:3200 connection refused`
- `tenant-id not found`

Possible reasons for these errors are:

- The querier isn't connected to the query-frontend. Check the value of the metric `cortex_query_frontend_connected_clients` exposed by the query-frontend.
  It should be > `0`, indicating querier connections with the query-frontend.
- Grafana Tempo data source isn't configured to pass `tenant-id` in the `Authorization` header (multi-tenant deployments only).
- Not connected to Tempo Querier correctly.
- Insufficient permissions.

### Solutions

To fix connection issues:

  - If the queriers aren't connected to the query-frontend, check the following section in the querier configuration and verify the query-frontend address.

    ```yaml
    querier:
      frontend_worker:
        frontend_address: query-frontend-discovery.default.svc.cluster.local:9095
    ```
  - Validate the Grafana data source configuration and debug network issues between Grafana and Tempo.

To fix an insufficient permissions issue:

  - Verify that the querier has the `LIST` and `GET` permissions on the bucket.
