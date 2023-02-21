# Datadog Exporter

| Status                   |                  |
| ------------------------ |------------------|
| Stability                | traces [beta]    |
|                          | metrics [beta]   |
|                          | logs [alpha]     |
| Supported pipeline types | traces, metrics, logs|
| Distributions            | [contrib], [AWS] |

> Please review the Collector's [security documentation](https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/security-best-practices.md), which contains recommendations on securing sensitive information such as the API key required by this exporter.

Visit the [official documentation](https://docs.datadoghq.com/tracing/trace_collection/open_standards/otel_collector_datadog_exporter/) for usage instructions.

## FAQs

### Why am I getting errors 413 - Request Entity Too Large, how do I fix it?

This error indicates the payload size sent by the Datadog exporter exceeds the size limit (see previous examples https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/16834, https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/17566).

This is usually caused by the pipeline batching too many telemetry data before sending to the Datadog exporter. To fix that, try lowering `send_batch_size` and `send_batch_max_size` in your batchprocessor config. You might want to have a separate batch processor dedicated for datadog exporter if other exporters expect a larger batch size, e.g.
```
processors:
  batch:  # To be used by other exporters
    timeout: 1s
    # Default value for send_batch_size is 8192
  batch/datadog:
    send_batch_max_size: 100
    send_batch_size: 10
    timeout: 10s
...
service:
  pipelines:
    metrics:
      receivers: ...
      processors: [batch/datadog]
      exporters: [datadog]
```

The exact values for `send_batch_size` and `send_batch_max_size` depends on your specific workload. Also note that, Datadog intake has different payload size limits for the 3 signal types:
- Trace intake: 3.2MB
- Log intake: https://docs.datadoghq.com/api/latest/logs/
- Metrics V2 intake: https://docs.datadoghq.com/api/latest/metrics/#submit-metrics


[beta]:https://github.com/open-telemetry/opentelemetry-collector#beta
[alpha]:https://github.com/open-telemetry/opentelemetry-collector#alpha
[contrib]:https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-contrib
[AWS]:https://aws-otel.github.io/docs/partners/datadog
