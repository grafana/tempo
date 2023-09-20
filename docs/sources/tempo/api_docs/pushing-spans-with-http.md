---
title: Push spans with HTTP
description: Learn a basic technique for pushing spans with HTTP and JSON
aliases:
- /docs/tempo/latest/guides/pushing-spans-with-http/
---

# Push spans with HTTP

Sometimes using a tracing system is intimidating because it seems like you need complex application instrumentation
or a span ingestion pipeline in order to push spans.  This guide aims to show an extremely basic technique for
pushing spans with HTTP/JSON from a Bash script using the [OpenTelemetry](https://opentelemetry.io/docs/specs/otlp/) receiver.

## Before you begin

This procedure uses an example Docker Compose setup to run Tempo, so you do not need an existing installation. The Docker image also includes a Grafana container which will allow us to visualize traces.

To use this procedure, you need to have Docker and `docker compose` installed.


## Start Tempo using the quick start

Use the instructions in the [Quick start for Tempo documentation]({{< relref "../getting-started/docker-example" >}}) to start a local instance of Tempo and Grafana.

## Push spans with OTLP

Now that Tempo is running and listening on port 4318 for [OTLP spans](https://opentelemetry.io/docs/specs/otlp/#otlphttp), let’s push a span to it using `curl`.

Before you can use this example, you need to update the start and end time as instructed.

```bash
curl -X POST -H 'Content-Type: application/json' http://localhost:4318/v1/traces -d '
{
	"resourceSpans": [{
    	"resource": {
        	"attributes": [{
            	"key": "service.name",
            	"value": {
                	"stringValue": "my.service"
            	}
        	}]
    	},
    	"scopeSpans": [{
        	"scope": {
            	"name": "my.library",
            	"version": "1.0.0",
            	"attributes": [{
                	"key": "my.scope.attribute",
                	"value": {
                    	"stringValue": "some scope attribute"
                	}
            	}]
        	},
        	"spans": [
        	{
            	"traceId": "5B8EFFF798038103D269B633813FC700",
            	"spanId": "EEE19B7EC3C1B100",
            	"name": "I am a span!",
            	"startTimeUnixNano": 1689969302000000000,
            	"endTimeUnixNano": 1689970000000000000,
            	"kind": 2,
            	"attributes": [
            	{
                	"key": "my.span.attr",
                	"value": {
                    	"stringValue": "some value"
                	}
            	}]
        	}]
    	}]
	}]
}'
```

Note that the `startTimeUnixNano` field is in nanoseconds and can be obtained by any tool that provides the epoch date in nanoseconds (for example, under Linux, `date +%s%8N`). The `endTimeUnixNano` field is also in nanoseconds, where 100000000 nanoseconds is 100 milliseconds.

1. Copy and paste the curl command into a text editor.

1. Replace `startTimeUnixNano` and `endTimeUnixNano` with current values for the last 24 hours to allow you to search for them using a 24 hour relative time range. You can get this in seconds and milliseconds from the following link.
Multiple the milliseconds value by 1,000,000 to turn it into nanoseconds. You can do this from a bash terminal with:

   ```bash
   echo $((<epochTimeMilliseconds> * 1000000))
   ```

1. Copy the updated curl command to a terminal window and run it.

1. View the trace in Grafana:
   1. Open a browser window to http://localhost:3000.
   1. Open the **Explorer** page and select the Tempo data source.
   1. Select the **Search** query type.
   1. Select **Run query** to list available traces.
   1. Select the trace ID (yellow box) to view details about the trace and its spans.

![Using the TraceQL query builder on Explore to view pushed trace in Grafana.](/static/img/docs/tempo/push-spans-search-span-grafana.png "View the span in Grafana")

## Retrieve traces

The easiest way to get the trace is to execute a simple curl command to Tempo.  The returned format is [OTLP](https://github.com/open-telemetry/opentelemetry-proto/blob/main/opentelemetry/proto/trace/v1/trace.proto).

1. Replace the trace ID in the `curl` command with the trace ID that was generated from the push. This information is is in the data that's sent with the `curl`. You could use Grafana’s Explorer page to find this, as shown in the previous section.

```bash
curl http://localhost:3200/api/traces/5b8efff798038103d269b633813fc700

{"batches":[{"resource":{"attributes":[{"key":"service.name","value":{"stringValue":"my.service"}}]},"scopeSpans":[{"scope":{"name":"my.library","version":"1.0.0"},"spans":[{"traceId":"W47/95gDgQPSabYzgT/HAA==","spanId":"7uGbfsPBsQA=","name":"I am a span!","kind":"SPAN_KIND_SERVER","startTimeUnixNano":"1689969302000000000","endTimeUnixNano":"1689970000000000000","attributes":[{"key":"my.span.attr","value":{"stringValue":"some value"}}],"status":{}}]}]}]}
```

1. Copy and paste the updated `curl` command into a terminal window.

### Use TraceQL to search for a trace

Alternatively, you can also use [TraceQL]({{< relref "../traceql" >}}) to search for the trace that was pushed.
You can search by using the unique trace attributes that were set:

```bash
curl -G -s http://localhost:3200/api/search --data-urlencode 'q={ .service.name = "my.service" }'

{"traces":[{"traceID":"5b8efff798038103d269b633813fc700","rootServiceName":"my.service","rootTraceName":"I am a span!","startTimeUnixNano":"1694718625557000000","durationMs":10000,"spanSet":{"spans":[{"spanID":"eee19b7ec3c1b100","startTimeUnixNano":"1694718625557000000","durationNanos":"10000000000","attributes":[{"key":"service.name","value":{"stringValue":"my.service"}}]}],"matched":1},"spanSets":[{"spans":[{"spanID":"eee19b7ec3c1b100","startTimeUnixNano":"1694718625557000000","durationNanos":"10000000000","attributes":[{"key":"service.name","value":{"stringValue":"my.service"}}]}],"matched":1}]}],"metrics":{"inspectedBytes":"292781","completedJobs":1,"totalJobs":1}}
```

To format this in a more human-readable output, consider using a [tool such as `jq`](https://jqlang.github.io/jq/), which lets you to run the same `curl` command and pipe it to `jq` to format the block. For example:

```bash
curl -G -s http://localhost:3200/api/search --data-urlencode 'q={ .service.name = "my.service" }' | jq
```

## Spans from everything!

Tracing is not limited to enterprise languages with complex frameworks.  As you can see it's easy to store and track events from your js, python or bash scripts.
You can use Tempo/distributed tracing today to trace CI pipelines, long running bash processes, python data processing flows, or anything else
you can think of.

Happy tracing!
