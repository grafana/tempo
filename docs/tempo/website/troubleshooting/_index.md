---
title: Troubleshooting
weight: 550
---

# Troubleshooting missing traces

This topic helps with day zero operational issues that may come up when getting started with Tempo. It walks through debugging each part of the ingestion and query pipeline to drill down and diagnose issues. 

## 1. Problem - I am unable to see any of my traces in Tempo

>** Potential causes**
- There could be issues in ingestion of the data into Tempo, that is, spans are either not being sent correctly to Tempo or they are not getting sampled.
- There could be issues querying for traces that have been received by Tempo.

### Diagnosing and fixing ingestion issues

Check whether the application spans are actually reaching Tempo. The following metrics help determine this
- `tempo_distributor_spans_received_total` 
- `tempodb_blocklist_length`

The value of both metrics should be greater than `0` within a few minutes of the application spinning up.
You can check both metrics using -
- The metrics page exposed from Tempo at `http://<tempo-address>:<tempo-http-port>/metrics`, or
- In Prometheus, if it is being used to scrape metrics
 
### Issue 1 - `tempo_distributor_spans_received_total` is 0

If the value of `tempo_distributor_spans_received_total` is 0, possible reasons are:
- Use of incorrect protocol/port combination while initializing the tracer in the application.
- Tracing records not getting picked up to send to Tempo by the internal sampler.
- Application is running inside docker and sending traces to an incorrect endpoint.

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

### Issue 2 - `tempodb_blocklist_length` is 0

If the value of `tempodb_blocklist_length` is 0, the possible reason is -
- Insufficient permissions to write to the back-end.

#### Solution

- Log in to GCS/S3 to check if the blocks are present in the bucket. If you are using the local backend, look inside `/tmp/local/traces` to see if the blocks are present.
- If blocks are missing, then it might indicate permission problems. Tempo Ingester requires LIST, GET and PUT permissions on the bucket. Ensure the container running the ingester has these permissions provided.
- If you still see nothing, please [file an issue](https://github.com/grafana/tempo/issues/new/choose) on the Tempo repository with a description and the configuration used.

### Diagnosing and fixing issues with querying traces
If you have determined that data has been ingested correctly into Tempo, then it is time to investigate possible issues with querying the data.

The presence of the following errors in the Tempo Querier log files may explain why traces are missing:

- `no org id`
- `could not dial 10.X.X.X:3100 connection refused`
- `tenant-id not found`

Possible reasons for the above errors are:
- Not connected to Tempo Querier correctly
- Insufficient permissions


#### Solution

- Fixing connection issues
  - In case the application is not connected to Tempo Querier correctly, update the `backend.yaml` configuration file so that it is attempting to connect to the right port of the querier.
- Fixing insufficient permissions issue
  - Verify that the Querier has the LIST and GET permissions on the bucket.


## 2. Problem - Some of my traces are missing in Tempo

This could happen because of a number of reasons and some have been detailed in this blog post -
[Where did all my spans go? A guide to diagnosing dropped spans in Jaeger distributed tracing](https://grafana.com/blog/2020/07/09/where-did-all-my-spans-go-a-guide-to-diagnosing-dropped-spans-in-jaeger-distributed-tracing/).

### Diagnosing the issue 

If the pipeline is not reporting any dropped spans, check whether application spans are being dropped by Tempo. The following metrics help determine this -
- `tempo_receiver_refused_spans`. The value of `tempo_receiver_refused_spans` should be 0.
If the value of `tempo_receiver_refused_spans` is greater than 0, then the possible reason is the application spans are being dropped due to rate limiting.

#### Solution
- The rate limiting may be appropriate and does not need to be fixed. The metric simply explained the cause of the missing spans, and there is nothing more to be done.
- If more ingestion volume is needed, increase the configuration for the rate limiting, by adding this CLI flag to Tempo at startup - https://github.com/grafana/tempo/blob/78f3554ca30bd5a4dec01629b8b7b2b0b2b489be/modules/overrides/limits.go#L42

## 3. Problem: Getting error message ‘Too many jobs in the queue’
You may see this error if the compactor isn’t running and the blocklist size has exploded. 
Possible reasons why the compactor may not be running are:

- Insufficient permissions.
- Compactor sitting idle because no block is hashing to it.
- Incorrect configuration settings.

### Diagnosing the issue
- Check metric `tempodb_compaction_bytes_written`
If this is greater than zero (0), it means the compactor is running and writing to the backend.
- Check metric `tempodb_compaction_errors_total`
If this metric is greater than zero (0), it likely means there are permission related issues.

#### Solutions
- Verify that the Compactor has the LIST, GET, PUT, and DELETE permissions on the bucket.
  - If these permissions are missing, assign them to the compactor container.
- If there’s a compactor sitting idle while others are running, port-forward to the compactor’s http endpoint. Then go to `/compactor/ring` and click **Forget** on the inactive compactor.
- Check the following configuration parameters to ensure that there are correct settings:
  - `traces_per_block` to determine when the ingester cuts blocks.
  - `max_compaction_objects` to determine the max number of objects in a compacted block (this should be higher than `tracer_per_block` for the compactor to actually combine multiple blocks together). This can run up to `100,000`
  - `retention_duration` for how long traces should be retained in the backend.
