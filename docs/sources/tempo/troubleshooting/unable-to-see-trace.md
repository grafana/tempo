---
title: Unable to find traces
description: Troubleshoot missing traces
weight: 473
aliases:
- ../operations/troubleshooting/missing-trace/
- ../operations/troubleshooting/unable-to-see-trace/
---

# Unable to find traces

The two main causes of missing traces are:

- Issues in ingestion of the data into Tempo. Spans are either not being sent correctly to Tempo or they are not getting sampled.
- Issues querying for traces that have been received by Tempo.

## Section 1: Diagnose and fix ingestion issues

The first step is to check whether the application spans are actually reaching Tempo.

Add the following flag to the distributor container - [`distributor.log_received_spans.enabled`](https://github.com/grafana/tempo/blob/57da4f3fd5d2966e13a39d27dbed4342af6a857a/modules/distributor/config.go#L55).

This flag enables debug logging of all the traces received by the distributor. These logs can help check if Tempo is receiving any traces at all.

You can also check the following metrics:

- `tempo_distributor_spans_received_total`
- `tempo_ingester_traces_created_total`

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

## Case 2 - tempo_ingester_traces_created_total is 0

If the value of `tempo_ingester_traces_created_total` is 0, this can indicate network issues between distributors and ingesters.

Checking the metric `tempo_request_duration_seconds_count{route='/tempopb.Pusher/Push'}` exposed from the ingester which indicates that it's receiving ingestion requests from the distributor.

### Solution

Check logs of distributors for a message like `msg="pusher failed to consume trace data" err="DoBatch: IngesterCount <= 0"`.
This is likely because no ingester is joining the gossip ring, make sure the same gossip ring address is supplied to the distributors and ingesters.

## Diagnose and fix sampling and limits issues

If you are able to query some traces in Tempo but not others, you have come to the right section.

This could happen because of a number of reasons and some have been detailed in this blog post:
[Where did all my spans go? A guide to diagnosing dropped spans in Jaeger distributed tracing](/blog/2020/07/09/where-did-all-my-spans-go-a-guide-to-diagnosing-dropped-spans-in-jaeger-distributed-tracing/).
This is useful if you are using the Jaeger Agent.

If you are using Grafana Agent, continue reading the following section for metrics to monitor.

### Diagnose the issue

Check if the pipeline is dropping spans. The following metrics on Grafana Agent help determine this:

- `tempo_exporter_send_failed_spans`. The value of this metric should be `0`.
- `tempo_receiver_refused_spans`. This value of this metric should be `0`.
- `tempo_processor_dropped_spans`. The value of this metric should be `0`.

If the pipeline isn't reporting any dropped spans, check whether application spans are being dropped by Tempo. The following metrics help determine this:

- `tempo_receiver_refused_spans`. The value of `tempo_receiver_refused_spans` should be `0`.

  Grafana Agent and Tempo share the same metric. Make sure to check the value of the metric from both services.
  If the value of `tempo_receiver_refused_spans` is greater than 0, then the possible reason is the application spans are being dropped due to rate limiting.

### Solutions

- If the pipeline (Grafana Agent) is found to be dropping spans, the deployment may need to be scaled up. Look for a message like `too few agents compared to the ingestion rate` in the agent logs.
- There might also be issues with connectivity to Tempo backend, check the agent for logs like `error sending batch, will retry` and make sure the Tempo endpoint and credentials are correctly configured.
- If Tempo is found to be dropping spans, then the possible reason is the application spans are being dropped due to rate limiting.
  The rate limiting may be appropriate and does not need to be fixed. The metric simply explained the cause of the missing spans, and there is nothing more to be done.
- If more ingestion volume is needed, increase the configuration for the rate limiting, by adding this CLI flag to Tempo at startup - https://github.com/grafana/tempo/blob/78f3554ca30bd5a4dec01629b8b7b2b0b2b489be/modules/overrides/limits.go#L42

{{< admonition type="note" >}}
Check the [ingestion limits page]({{< relref "../configuration#ingestion-limits" >}}) for further information on limits.
{{% /admonition %}}

## Section 3: Diagnose and fix issues with querying traces

If you have determined that data has been ingested correctly into Tempo, then it's time to investigate possible issues with querying the data.

Check the logs of the query-frontend. The query-frontend pod runs with two containers (query-frontend and query), so lets use the following command to view query-frontend logs -

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
  It should be > `0`, which indicates that queriers are connected to the query-frontend.
- Grafana Tempo data source isn't configured to pass `tenant-id` in the `Authorization` header (only applicable to multi-tenant deployments).
- Not connected to Tempo Querier correctly.
- Insufficient permissions.

### Solutions

To fix connection issues:

  - If the queriers aren't connected to the query-frontend, check the following section in the querier configuration and make sure the address of the query-frontend is correct.

    ```yaml
    querier:
      frontend_worker:
        frontend_address: query-frontend-discovery.default.svc.cluster.local:9095
    ```
<!-- >  - Verify the `backend.yaml` configuration file present on the Tempo Query container and make sure it is attempting to connect to the right port of the query frontend.
    **Note** this is only relevant for [Grafana 7.4.x and before](https://grafana.com/docs/tempo/latest/configuration/querying/#grafana-74x).
    -->
  - Confirm that the Grafana data source is configured correctly and debug network issues between Grafana and Tempo.

To fix an insufficient permissions issue:

  - Verify that the querier has the `LIST` and `GET` permissions on the bucket.
