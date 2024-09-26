---
description: Learn how to use the metrics summary API in Tempo
keywords:
  - Metrics summary
  - API
title: Metrics summary API
weight: 600
---

# Metrics summary API

{{< admonition type="warning" >}}
The Metrics summary API is an [experimental feature](/docs/release-life-cycle) that is disabled by default. To enable it, adjust your configuration as suggested below.
{{% /admonition %}}

This document explains how to use the metrics summary API in Tempo.
This API returns RED metrics (span count, erroring span count, and latency information) for `kind=server` spans sent to Tempo in the last hour, grouped by a user-specified attribute.

{{< youtube id="g97CjKOZqT4" >}}

## Configuration

To enable the experimental metrics summary API, you must turn on the local blocks processor in the metrics generator.
Be aware that the generator uses considerably more resources, including disk space, if it's enabled:

```yaml
overrides:
  defaults:
    metrics_generator:
      processors: [..., 'local-blocks']
```

## Request

To make a request to this API, use the following endpoint on the query-frontend:

```
GET http://<tempo>/api/metrics/summary
```

### Query Parameters

All query parameters must be URL-encoded to preserve non-URL-safe characters in the query such as `&`.

| Name      | Examples                                                                                         | Definition                                                                                                                                | Required? |
| --------- | ------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------- | --------- |
| `q`       | `{ resource.service.name = "foo" && span.http.status_code != 200 }`                              | The TraceQL query with full syntax. All spans matching this query are included in the calculations. Any valid TraceQL query is supported. | Yes       |
| `groupBy` | `name` <br /> `.foo` <br/> `resource.namespace` <br/> `span.http.url,span.http.status_code` <br> | One or more TraceQL values to group by. Any valid intrinsic or attribute with scope. To group by multiple values use a comma-delimited list.   | Yes       |
| `start `  | 1672549200                                                                                       | Start of time range in Unix seconds. If not specified, then all recent data is queried.                                                  | No        |
| `end`     | 1672549200                                                                                       | End of the time range in Unix seconds. If not specified, then all recent data is queried.                                                | No        |

Example:

```bash
curl "$URL/api/metrics/summary" --data-urlencode 'q={resource.service.name="checkout-service"}' --data-urlencode 'groupBy=name'
```

## Response

The Tempo response is a `SpanMetricsSummary` object defined in [tempo.proto](https://github.com/grafana/tempo/blob/main/pkg/tempopb/tempo.proto#L234), relevant section pasted below:

```
message SpanMetricsSummaryResponse {
  repeated SpanMetricsSummary summaries = 1;
}

message SpanMetricsSummary {
  uint64 spanCount = 1;
  uint64 errorSpanCount = 2;
  TraceQLStatic static = 3;
  uint64 p99 = 4;
  uint64 p95 = 5;
  uint64 p90 = 6;
  uint64 p50 = 7;
}

message TraceQLStatic {
  int32 type = 1;
  int64 n = 2;
  double f = 3;
  string s = 4;
  bool b = 5;
  uint64 d = 6;
  int32 status = 7;
  int32 kind = 8;
}

```

The response is returned as JSON following [standard protobuf->JSON mapping rules](https://protobuf.dev/programming-guides/proto3/#json).

{{< admonition type="note" >}}
The `uint64` fields cannot be fully expressed by JSON numeric values so the fields are serialized as strings.
{{% /admonition %}}

Example:

```javascript
{
   "summaries": [
       {
           "spanCount": "20",
           "series" : [
               {
                   "key": ".attr1",
                   "value": {
                       "type": 5,
                       "s": "foo"
                   },
               },
               ...
           ],
           "p99": "68719476736",
           "p95": "1073741824",
           "p90": "1017990479",
           "p50": "664499239"
       },
```

| Field             | Notes                                                                                                                                                                                                                                                                                               |
| ----------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `summaries`       | The list of metrics per group.                                                                                                                                                                                                                                                                     |
| `.spanCount`      | Number of spans in this group.                                                                                                                                                                                                                                                                     |
| `.errorSpanCount` | Number of spans with `status`=`error`. (This field will not be present if the value is `0`.)                                                                                                                                                                                                        |
| `.series`         | The unique values for this group. A key/value pair will be returned for each entry in `groupBy`.                                                                                                                                                                                                   |
| `.key`            | Key name.                                                                                                                                                                                                                                                                                          |
| `.value`          | Value with TraceQL underlying type.                                                                                                                                                                                                                                                                |
| `.type`           | Data type `enum`` defined [here](https://github.com/grafana/tempo/blob/main/pkg/traceql/enum_statics.go#L8) (This field will not be present if the value is `0`.) <br/>0 = `nil`<br/>3 = `integer`<br/> 4 = `float` <br/> 5 = `string`<br/> 6 = `bool`<br/> 7 = `duration`<br/> 8 = span status<br/> 9 = span kind |
| `.n`              | Populated if this is an integer value.                                                                                                                                                                                                                                                             |
| `.s`              | Populated if this is a string value.                                                                                                                                                                                                                                                               |
| `.f`              | Populated if this is a float value.                                                                                                                                                                                                                                                                |
| `.b`              | Populated if this is a boolean value.                                                                                                                                                                                                                                                              |
| `.d`              | Populated if this is a duration value.                                                                                                                                                                                                                                                             |
| `.status`         | Populated if this is a span status value.                                                                                                                                                                                                                                                          |
| `.kind`           | Populated if this is a span kind value.                                                                                                                                                                                                                                                            |
| `.p99`            | The p99 latency of this group in nanoseconds.                                                                                                                                                                                                                                                      |
| `.p95`            | The p95 latency of this group in nanoseconds.                                                                                                                                                                                                                                                      |
| `.p90`            | The p90 latency of this group in nanoseconds.                                                                                                                                                                                                                                                      |
| `.p50`            | The p50 latency of this group in nanoseconds.                                                                                                                                                                                                                                                      |
