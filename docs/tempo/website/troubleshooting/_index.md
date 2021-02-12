---
title: Troubleshooting
weight: 470
---

# Troubleshooting missing traces
This topic helps with day zero operational issues that may come up when getting started with Tempo. It walks through debugging each part of the ingestion and query pipeline to drill down and diagnose issues. 

## Problem 1. I am unable to see any of my traces in Tempo
** Potential causes**
- There could be issues in ingestion of the data into Tempo, that is, spans are either not being sent correctly to Tempo or they are not getting sampled.
- There could be issues querying for traces that have been received by Tempo.

### Diagnosing and fixing ingestion issues
Check whether the application spans are actually reaching Tempo. The following metrics help determine this
- `tempo_distributor_spans_received_total`
- `tempo_ingester_traces_created_total`

The value of both metrics should be greater than `0` within a few minutes of the application spinning up.
You can check both metrics using -
- The metrics page exposed from Tempo at `http://<tempo-address>:<tempo-http-port>/metrics`, or
- In Prometheus, if it is being used to scrape metrics
 
### Issue 1 - `tempo_distributor_spans_received_total` is 0
If the value of `tempo_distributor_spans_received_total` is 0, possible reasons are:
- Use of incorrect protocol/port combination while initializing the tracer in the application.
- Tracing records not getting picked up to send to Tempo by the internal sampler.
- Application is running inside docker and sending traces to an incorrect endpoint.

Receiver specific traffic information can also be obtained using `tempo_receiver_accepted_spans` which has a label for the receiver (protocol used for ingestion. Ex: `jaeger-thrift`).

#### Solutions

##### Fixing protocol/port problems
- Find out which communication protocol is being used by the application to emit traces. This is unique to every client SDK. For instance: Jaeger Golang Client uses `Thrift Compact over UDP` by default.
- Check the list of supported protocols and their ports and ensure that the correct combination is being used. You will find the list of supported protocols and ports here: https://grafana.com/docs/tempo/latest/getting-started/#step-1-spin-up-tempo-backend

##### Fixing sampling issues
- These issues can be tricky to determine because most SDKs use a probabilistic sampler by default. This may lead to just one in a 1000 records being picked up.
- Check the sampling configuration of the tracer being initialized in the application and make sure it has a high sampling rate.
- Some clients also provide metrics on the number of spans reported from the application, for example `jaeger_tracer_reporter_spans_total`. Check the value of that metric if available and make sure it is greater than zero.
- Another way to diagnose this problem would be to generate lots and lots of traces to see if some records make their way to Tempo.

##### Fixing incorrect endpoint issue
- If the application is also running inside docker, make sure the application is sending traces to the correct endpoint (`tempo:<receiver-port>`).

### Issue 2 - `tempo_ingester_traces_created_total` is 0
If the value of `tempo_ingester_traces_created_total` is 0, the possible reason is -
- Network issues between distributors and ingesters.

This can also be confirmed by checking the metric `tempo_request_duration_seconds_count{route='/tempopb.Pusher/Push'}` exposed from the ingester which indicates that it is receiving ingestion requests from the distributor.

#### Solution
- Check logs of distributors for a message like `msg="pusher failed to consume trace data" err="DoBatch: IngesterCount <= 0"`.
  This is likely because no ingester is joining the gossip ring, make sure the same gossip ring address is supplied to the distributors and ingesters.

### Diagnosing and fixing issues with querying traces
If you have determined that data has been ingested correctly into Tempo, then it is time to investigate possible issues with querying the data.

Check the logs of the Tempo Query Frontend. The Query Frontend pod runs with two containers (Query Frontend & Tempo Query), so lets use the following command to view Query Frontend logs -

```console
kubectl logs -f pod/query-frontend-xxxxx -c query-frontend
```

The presence of the following errors in the log may explain issues with querying traces:

- `level=info ts=XXXXXXX caller=frontend.go:63 method=GET traceID=XXXXXXXXX url=/api/traces/XXXXXXXXX duration=5m41.729449877s status=500`
- `no org id`
- `could not dial 10.X.X.X:3100 connection refused`
- `tenant-id not found`

Possible reasons for the above errors are:
- Tempo Querier not connected to Tempo Query Frontend. Check the value of the metric `cortex_query_frontend_connected_clients` exposed by the Query Frontend.
  It should be > 0, which indicates that Queriers are connected to the Query Frontend.
- Grafana Tempo Datasource not configured to pass tenant-id in Authorization header (only applicable to multi-tenant deployments).
- Not connected to Tempo Querier correctly
- Insufficient permissions

#### Solution
- Fixing connection issues
  - In case we the queriers are not connected to the Query Frontend, check the following section in Querier configuration and make sure the address of the Query Frontend is correct
    ```
    querier:
      frontend_worker:
        frontend_address: query-frontend-discovery.default.svc.cluster.local:9095
    ```
  - Verify the `backend.yaml` configuration file present on the Tempo Query container and make sure it is attempting to connect to the right port of the query frontend.
  - Verify the Grafana Tempo Datasource configuration, and make sure it has the following settings configured:
    ```
    jsonData:
      httpHeaderName1: 'Authorization'
    secureJsonData:
      httpHeaderValue1: 'Bearer <tenant-id>'
    ```
- Fixing insufficient permissions issue
  - Verify that the Querier has the LIST and GET permissions on the bucket.


## Problem 2. Some of my traces are missing in Tempo
This could happen because of a number of reasons and some have been detailed in this blog post -
[Where did all my spans go? A guide to diagnosing dropped spans in Jaeger distributed tracing](https://grafana.com/blog/2020/07/09/where-did-all-my-spans-go-a-guide-to-diagnosing-dropped-spans-in-jaeger-distributed-tracing/).

### Diagnosing the issue 
If the pipeline is not reporting any dropped spans, check whether application spans are being dropped by Tempo. The following metrics help determine this -
- `tempo_receiver_refused_spans`. The value of `tempo_receiver_refused_spans` should be 0.
If the value of `tempo_receiver_refused_spans` is greater than 0, then the possible reason is the application spans are being dropped due to rate limiting.

#### Solution
- The rate limiting may be appropriate and does not need to be fixed. The metric simply explained the cause of the missing spans, and there is nothing more to be done.
- If more ingestion volume is needed, increase the configuration for the rate limiting, by adding this CLI flag to Tempo at startup - https://github.com/grafana/tempo/blob/78f3554ca30bd5a4dec01629b8b7b2b0b2b489be/modules/overrides/limits.go#L42

## Problem 3. Getting error message ‘Too many jobs in the queue’
You may see this error if the compactor isn’t running and the blocklist size has exploded. 
Possible reasons why the compactor may not be running are:

- Insufficient permissions.
- Compactor sitting idle because no block is hashing to it.
- Incorrect configuration settings.
### Diagnosing the issue
- Check metric `tempodb_compaction_bytes_written_total`
If this is greater than zero (0), it means the compactor is running and writing to the backend.
- Check metric `tempodb_compaction_errors_total`
If this metric is greater than zero (0), check the logs of the compactor for an error message.

#### Solutions
- Verify that the Compactor has the LIST, GET, PUT, and DELETE permissions on the bucket objects.
  - If these permissions are missing, assign them to the compactor container.
  - For detailed information, check - https://grafana.com/docs/tempo/latest/configuration/s3/#permissions
- If there’s a compactor sitting idle while others are running, port-forward to the compactor’s http endpoint. Then go to `/compactor/ring` and click **Forget** on the inactive compactor.
- Check the following configuration parameters to ensure that there are correct settings:
  - `max_block_bytes` to determine when the ingester cuts blocks. A good number is anywhere from 100MB to 2GB depending on the workload.
  - `max_compaction_objects` to determine the max number of objects in a compacted block. This should relatively high, generally in the millions.
  - `retention_duration` for how long traces should be retained in the backend.
