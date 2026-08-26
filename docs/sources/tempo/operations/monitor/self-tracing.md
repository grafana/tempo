---
title: Configure Tempo self-tracing
menuTitle: Configure self-tracing
description: Configure Tempo to export its own traces using OpenTelemetry or Jaeger environment variables.
weight: 40
---

# Configure Tempo self-tracing

Tempo uses the [OpenTelemetry SDK](https://github.com/open-telemetry/opentelemetry-go) to instrument itself.

Tempo detects which tracing configuration to use based on the environment variables you set:

- OpenTelemetry format uses the standard OTel environment variables.
- Jaeger format uses Jaeger-specific environment variables for backward compatibility.

Jaeger variables take precedence when both formats are set, and when no variables are set, Tempo doesn't export traces.

## Configure with OpenTelemetry environment variables

Set any of the following standard [OpenTelemetry SDK configuration](https://opentelemetry.io/docs/languages/sdk-configuration/otlp-exporter/) variables to enable self-tracing.

- `OTEL_EXPORTER_OTLP_ENDPOINT`: The OTLP endpoint URL. For example, `http://tempo:4318`.
- `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT`: The traces-specific endpoint URL. Overrides `OTEL_EXPORTER_OTLP_ENDPOINT`.
- `OTEL_TRACES_EXPORTER`: The traces exporter. Defaults to `otlp`. Set to `none` to propagate trace context without exporting traces.

The exporter uses OTLP over HTTP by default, set `OTEL_EXPORTER_OTLP_PROTOCOL` to change the protocol.

### Configure sampling

Configure sampling with the standard [`OTEL_TRACES_SAMPLER` and `OTEL_TRACES_SAMPLER_ARG`](https://opentelemetry.io/docs/languages/sdk-configuration/general/) environment variables:

- `always_on`: Sample all traces.
- `always_off`: Sample no traces.
- `traceidratio`: Sample by the trace ID ratio in `OTEL_TRACES_SAMPLER_ARG`.
- `parentbased_always_on` (default): Follow the parent span decision. Always sample root spans.
- `parentbased_always_off`: Follow the parent span decision. Never sample root spans.
- `parentbased_traceidratio`: Follow the parent span decision. Use the ratio for root spans.


You can use `jaeger_remote` to use Jaeger remote sampling, and `parentbased_jaeger_remote` to use parent-based Jaeger remote sampling.

The Jaeger remote samplers require `OTEL_TRACES_SAMPLER_ARG` in this format:

```none
endpoint=http://localhost:5778/sampling,pollingIntervalMs=5000,initialSamplingRate=0.25
```

### Configure context propagation

Configure trace context propagation with `OTEL_PROPAGATORS`:

- `tracecontext`: W3C Trace Context.
- `baggage`: W3C Baggage.
- `jaeger`: Jaeger propagation.
- `none`: No propagation.

The default is `tracecontext`, `baggage`, and `jaeger`.

### Force sampling

To force sampling of a single request, set the `Jaeger-Debug-Id` HTTP header on the request.
Tempo samples the request regardless of the sampler configuration and adds the header value as the `jaeger-debug-id` span attribute.

NOTE: this requires configuration with the `jaeger` propagator active.

## Configure with Jaeger environment variables

Set any of the following variables to enable self-tracing in Jaeger format:

- `JAEGER_AGENT_HOST`: Jaeger agent hostname.
- `JAEGER_ENDPOINT`: Jaeger collector endpoint URL.
- `JAEGER_SAMPLER_MANAGER_HOST_PORT`: Jaeger sampler manager endpoint.

### Configure Jaeger sampling

Configure sampling with the following Jaeger environment variables:

- `JAEGER_SAMPLER_TYPE`: One of `const`, `probabilistic`, or `remote`.
- `JAEGER_SAMPLER_PARAM`: Sampling parameter value.
- `JAEGER_SAMPLING_ENDPOINT`: Remote sampling endpoint. Overrides `JAEGER_SAMPLER_MANAGER_HOST_PORT`.

When a remote sampling endpoint is set and `JAEGER_SAMPLER_TYPE` isn't, Tempo uses the `remote` sampler.

### Additional Jaeger environment variables

- `JAEGER_AGENT_PORT`: Jaeger agent port. Defaults to `6831`.
- `JAEGER_TAGS`: Resource attributes added to reported traces, in the format `key1=value1,key2=value2`.
- `JAEGER_REPORTER_MAX_QUEUE_SIZE`: Maximum queue size of the span batch processor.


## Configure the service name

Tempo reports traces with the service name with `tempo-<target>` format, where `<target>` is the value of the `-target` flag.

Set `OTEL_SERVICE_NAME` or `JAEGER_SERVICE_NAME` to override it. `JAEGER_SERVICE_NAME` takes precedence when both are set.

Use `OTEL_RESOURCE_ATTRIBUTES` to add custom resource attributes to the reported traces.

## Enable span profiling

Span profiling links trace spans to the profiles collected while those spans were executing. 

When enabled, Tempo attaches pprof goroutine labels (`span_id`, `span_name`) to its spans and adds a `pyroscope.profile.id` attribute to root spans. These labels are embedded in the profile data, and profiles scraped from `/debug/pprof` include them.

Enable span profiling in your configuration file or with the `--span-profiling` CLI flag:

```yaml
span_profiling: true
```

{{< admonition type="note" >}}
Span profiling only takes effect when self-tracing is enabled.
{{< /admonition >}}

For more information, refer to [command-line flags](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/command-line-flags/#span-profiling) and [Link tracing and profiling with Span Profiles](https://grafana.com/docs/pyroscope/latest/configure-client/trace-span-profiles/).
