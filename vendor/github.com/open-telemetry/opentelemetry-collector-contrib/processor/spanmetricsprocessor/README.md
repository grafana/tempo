# Span Metrics Processor

**Note:** Currently experimental and subject to breaking changes (e.g. change from processor to exporter/translator component).
See: https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/403.

Aggregates Request, Error and Duration (R.E.D) metrics from span data.

**Request** counts are computed as the number of spans seen per unique set of dimensions, including Errors.
For example, the following metric shows 142 calls:
```
calls{http_method="GET",http_status_code="200",operation="/Address",service_name="shippingservice",span_kind="SPAN_KIND_SERVER",status_code="STATUS_CODE_UNSET"} 142
```
Multiple metrics can be aggregated if, for instance, a user wishes to view call counts just on `service_name` and `operation`.

**Error** counts are computed from the Request counts which have an "Error" Status Code metric dimension.
For example, the following metric indicates 220 errors:
```
calls{http_method="GET",http_status_code="503",operation="/checkout",service_name="frontend",span_kind="SPAN_KIND_CLIENT",status_code="STATUS_CODE_ERROR"} 220
```

**Duration** is computed from the difference between the span start and end times and inserted into the
relevant latency histogram time bucket for each unique set dimensions.
For example, the following latency buckets indicate the vast majority of spans (9K) have a 100ms latency:
```
latency_bucket{http_method="GET",http_status_code="200",label1="value1",operation="/Address",service_name="shippingservice",span_kind="SPAN_KIND_SERVER",status_code="STATUS_CODE_UNSET",le="2"} 327
latency_bucket{http_method="GET",http_status_code="200",label1="value1",operation="/Address",service_name="shippingservice",span_kind="SPAN_KIND_SERVER",status_code="STATUS_CODE_UNSET",le="6"} 751
latency_bucket{http_method="GET",http_status_code="200",label1="value1",operation="/Address",service_name="shippingservice",span_kind="SPAN_KIND_SERVER",status_code="STATUS_CODE_UNSET",le="10"} 1195
latency_bucket{http_method="GET",http_status_code="200",label1="value1",operation="/Address",service_name="shippingservice",span_kind="SPAN_KIND_SERVER",status_code="STATUS_CODE_UNSET",le="100"} 10180
latency_bucket{http_method="GET",http_status_code="200",label1="value1",operation="/Address",service_name="shippingservice",span_kind="SPAN_KIND_SERVER",status_code="STATUS_CODE_UNSET",le="250"} 10180
...
```

Each metric will have _at least_ the following dimensions because they are common across all spans:
- Service name
- Operation
- Span kind
- Status code

This processor lets traces to continue through the pipeline unmodified.

The following settings are required:

- `metrics_exporter`: the name of the exporter that this processor will write metrics to. This exporter **must** be present in a pipeline.

The following settings can be optionally configured:

- `latency_histogram_buckets`: the list of durations defining the latency histogram buckets.
  - Default: `[2ms, 4ms, 6ms, 8ms, 10ms, 50ms, 100ms, 200ms, 400ms, 800ms, 1s, 1400ms, 2s, 5s, 10s, 15s]`
- `dimensions`: the list of dimensions to add together with the default dimensions defined above.
  
  Each additional dimension is defined with a `name` which is looked up in the span's collection of attributes or
  resource attributes (AKA process tags) such as `ip`, `host.name` or `region`.
  
  If the `name`d attribute is missing in the span, the optional provided `default` is used.
  
  If no `default` is provided, this dimension will be **omitted** from the metric.

## Examples

The following is a simple example usage of the spanmetrics processor.

For configuration examples on other use cases, please refer to [More Examples](#more-examples).

The full list of settings exposed for this processor are documented [here](./config.go).

```yaml
receivers:
  jaeger:
    protocols:
      thrift_http:
        endpoint: "0.0.0.0:14278"

  # Dummy receiver that's never used, because a pipeline is required to have one.
  otlp/spanmetrics:
    protocols:
      grpc:
        endpoint: "localhost:12345"

  otlp:
    protocols:
      grpc:
        endpoint: "localhost:55677"

processors:
  batch:
  spanmetrics:
    metrics_exporter: otlp/spanmetrics
    latency_histogram_buckets: [100us, 1ms, 2ms, 6ms, 10ms, 100ms, 250ms]
    dimensions:
      - name: http.method
        default: GET
      - name: http.status_code

exporters:
  jaeger:
    endpoint: localhost:14250

  otlp/spanmetrics:
    endpoint: "localhost:55677"
    tls:
      insecure: true

  prometheus:
    endpoint: "0.0.0.0:8889"

service:
  pipelines:
    traces:
      receivers: [jaeger]
      processors: [spanmetrics, batch]
      exporters: [jaeger]

    # The exporter name must match the metrics_exporter name.
    # The receiver is just a dummy and never used; added to pass validation requiring at least one receiver in a pipeline.
    metrics/spanmetrics:
      receivers: [otlp/spanmetrics]
      exporters: [otlp/spanmetrics]

    metrics:
      receivers: [otlp]
      exporters: [prometheus]
```

### More Examples

For more example configuration covering various other use cases, please visit the [testdata directory](./testdata).
