---
title: Configure Tempo self-tracing
menuTitle: Configure self-tracing
description: Configure Tempo to export its own traces using OpenTelemetry or Jaeger environment variables.
weight: 40
---

# Configure Tempo self-tracing

Distributed tracing is a valuable tool for troubleshooting the behavior of Tempo in production.
Tempo uses the [OpenTelemetry SDK](https://github.com/open-telemetry/opentelemetry-go) to instrument its own read path and parts of its write path.

Tempo detects which tracing configuration to use based on the environment variables you set:

- OpenTelemetry format uses the standard OTel environment variables.
- Jaeger format uses Jaeger-specific environment variables for backward compatibility.

When variables for both formats are set, the Jaeger variables take precedence.
When no variables are set, Tempo doesn't export traces.

## Configure with OpenTelemetry environment variables

Set any of the following environment variables to enable self-tracing in OpenTelemetry format.
These variables follow the standard [OpenTelemetry SDK configuration](https://opentelemetry.io/docs/languages/sdk-configuration/otlp-exporter/).

- `OTEL_EXPORTER_OTLP_ENDPOINT`: The OTLP endpoint URL. For example, `http://tempo:4318`.
- `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT`: The traces-specific OTLP endpoint URL. This value overrides `OTEL_EXPORTER_OTLP_ENDPOINT`. For example, `http://tempo:4318/v1/traces`.
- `OTEL_TRACES_EXPORTER`: The traces exporter to use. Defaults to `otlp`. Set to `none` to propagate trace context without exporting traces.

The exporter uses OTLP over HTTP by default.
Set `OTEL_EXPORTER_OTLP_PROTOCOL` to change the protocol.

### Configure sampling

Configure sampling with the standard [`OTEL_TRACES_SAMPLER` and `OTEL_TRACES_SAMPLER_ARG`](https://opentelemetry.io/docs/languages/sdk-configuration/general/) environment variables.

`OTEL_TRACES_SAMPLER` supports the following values:

- `always_on`: Sample all traces.
- `always_off`: Sample no traces.
- `traceidratio`: Sample traces based on the trace ID ratio in `OTEL_TRACES_SAMPLER_ARG`.
- `parentbased_always_on`: Sample based on the parent span decision. Always sample root spans. This is the default.
- `parentbased_always_off`: Sample based on the parent span decision. Never sample root spans.
- `parentbased_traceidratio`: Sample based on the parent span decision. Use the ratio for root spans.
- `jaeger_remote`: Use Jaeger remote sampling.
- `parentbased_jaeger_remote`: Use parent-based Jaeger remote sampling.

The `jaeger_remote` and `parentbased_jaeger_remote` samplers require `OTEL_TRACES_SAMPLER_ARG` in the following format:

```none
endpoint=http://localhost:5778/sampling,pollingIntervalMs=5000,initialSamplingRate=0.25
```

### Configure propagation

Configure trace context propagation with `OTEL_PROPAGATORS`.
It supports the following values:

- `tracecontext`: W3C Trace Context.
- `baggage`: W3C Baggage.
- `jaeger`: Jaeger propagation.
- `none`: No propagation.

When `OTEL_PROPAGATORS` isn't set, Tempo uses the `tracecontext`, `baggage`, and `jaeger` propagators.

### Force sampling of a request

To force sampling of a single request, set the `Jaeger-Debug-Id` HTTP header on the request.
Tempo samples the request regardless of the sampler configuration and adds the header value as the `jaeger-debug-id` span attribute.
This only applies when self-tracing is configured with OpenTelemetry environment variables.

### Example OpenTelemetry configuration

```bash
# Basic OTLP configuration
export OTEL_EXPORTER_OTLP_ENDPOINT="http://tempo:4318"
export OTEL_TRACES_SAMPLER="always_on"

# Advanced configuration with Jaeger remote sampling
export OTEL_EXPORTER_OTLP_TRACES_ENDPOINT="http://tempo:4318/v1/traces"
export OTEL_TRACES_SAMPLER="parentbased_jaeger_remote"
export OTEL_TRACES_SAMPLER_ARG="endpoint=http://localhost:5778/sampling,pollingIntervalMs=5000,initialSamplingRate=0.25"
export OTEL_PROPAGATORS="tracecontext,baggage,jaeger"
```

## Configure with Jaeger environment variables

Set any of the following environment variables to enable self-tracing in Jaeger format:

- `JAEGER_AGENT_HOST`: Jaeger agent hostname.
- `JAEGER_ENDPOINT`: Jaeger collector endpoint URL.
- `JAEGER_SAMPLER_MANAGER_HOST_PORT`: Jaeger sampler manager endpoint.

These variables take precedence over the OpenTelemetry variables when both are set.

### Configure Jaeger sampling

Configure sampling with the following Jaeger environment variables:

- `JAEGER_SAMPLER_TYPE`: Sampling strategy. One of `const`, `probabilistic`, or `remote`.
- `JAEGER_SAMPLER_PARAM`: Sampling parameter value.
- `JAEGER_SAMPLING_ENDPOINT`: Remote sampling endpoint. Takes precedence over `JAEGER_SAMPLER_MANAGER_HOST_PORT`.

When a remote sampling endpoint is set and `JAEGER_SAMPLER_TYPE` isn't, Tempo uses the `remote` sampler.

### Additional Jaeger environment variables

- `JAEGER_AGENT_PORT`: Jaeger agent port. Defaults to `6831`.
- `JAEGER_TAGS`: Additional attributes to add to spans, in the format `key1=value1,key2=value2`.
- `JAEGER_REPORTER_MAX_QUEUE_SIZE`: Maximum queue size of the span batch processor.

### Example Jaeger configuration

```bash
# Basic agent configuration
export JAEGER_AGENT_HOST="jaeger-agent"
export JAEGER_SAMPLER_TYPE="const"
export JAEGER_SAMPLER_PARAM="1"

# Remote sampling configuration
export JAEGER_AGENT_HOST="jaeger-agent"
export JAEGER_SAMPLER_MANAGER_HOST_PORT="http://jaeger-agent:5778/sampling"
```

## Configure the service name

Tempo reports traces with the service name `tempo-<target>`, where `<target>` is the value of the `-target` flag.
Set `OTEL_SERVICE_NAME` or `JAEGER_SERVICE_NAME` to override the service name.
`OTEL_SERVICE_NAME` takes precedence when both are set.

Use `OTEL_RESOURCE_ATTRIBUTES` to add custom resource attributes to the reported traces.

## Enable span profiling

Span profiling links Tempo's trace spans to the profiles collected while those spans were executing.
Use it to jump from a slow span directly to the matching profile in Grafana Pyroscope.
When enabled, Tempo attaches pprof goroutine labels (`span_id`, `span_name`) to its OpenTelemetry spans and adds a `pyroscope.profile.id` attribute to root spans.
These labels are part of the profile data, so they are also present in profiles scraped from `/debug/pprof` endpoints.

Enable span profiling in your configuration file or with the `--span-profiling` CLI flag:

```yaml
span_profiling: true
```

{{< admonition type="note" >}}
Span profiling only takes effect when self-tracing is enabled.
Set the OpenTelemetry environment variables described on this page.
{{< /admonition >}}

For more information, refer to [command-line flags](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/command-line-flags/#span-profiling) and [Link tracing and profiling with Span Profiles](https://grafana.com/docs/pyroscope/latest/configure-client/trace-span-profiles/).
