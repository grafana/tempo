---
aliases:
  - ../../server_side_metrics/span_metrics/ # /docs/tempo/next/server_side_metrics/span_metrics/
  - ../../metrics-generator/span_metrics/ # /docs/tempo/next/metrics-generator/span_metrics/
title: Use the span metrics processor
menuTitle: Use metrics-generator
description: The span metrics processor generates metrics from ingested tracing data, including request, error, and duration (RED) metrics.
weight: 200
refs:
  cardinality:
    - pattern: /docs/tempo/
      destination: https://grafana.com/docs/tempo/<TEMPO_VERSION>/metrics-from-traces/metrics-generator/cardinality/
    - pattern: /docs/enterprise-traces/
      destination: https://grafana.com/docs/enterprise-traces/<ENTERPRISE_TRACES_VERSION>/metrics-generator/cardinality/
---

# Use the metrics-generator to create metrics from spans

Part of the metrics-generator, the span metrics processor generates metrics from ingested tracing data, including request, error, and duration (RED) metrics.

Span metrics generate three metrics:

- A counter that computes requests
- A histogram that tracks the distribution of durations of all requests
- A counter that tracks the total size of spans ingested

Span metrics are of particular interest if your system is not monitored with metrics,
but it has distributed tracing implemented.
You get out-of-the-box metrics from your tracing pipeline.

Even if you already have metrics, span metrics can provide in-depth monitoring of your system.
The generated metrics will show application level insight into your monitoring,
as far as tracing gets propagated through your applications.

Last but not least, span metrics lower the entry barrier for using [exemplars](https://grafana.com/docs/grafana/<GRAFANA_VERSION>/basics/exemplars/).
An exemplar is a specific trace representative of measurement taken in a given time interval.
Since traces and metrics co-exist in the metrics-generator,
exemplars can be automatically added, providing additional value to these metrics.

## How to run

To enable span metrics in Tempo or Grafana Enterprise Traces, enable the metrics generator and add an overrides section which enables the `span-metrics` processor.
Refer to [the configuration details](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration#metrics-generator).

If you want to enable metrics-generator for your Grafana Cloud account, refer to the [Metrics-generator in Grafana Cloud](https://grafana.com/docs/grafana-cloud/send-data/traces/metrics-generator/) documentation.

### Enabling specific metrics (subprocessors)

Instead of enabling all span metrics, you can enable individual metric types using subprocessors in the overrides configuration:

- `span-metrics-latency` - Enables only the `traces_spanmetrics_latency` histogram
- `span-metrics-count` - Enables only the `traces_spanmetrics_calls_total` counter
- `span-metrics-size` - Enables only the `traces_spanmetrics_size_total` counter

Example overrides configuration:

```yaml
overrides:
  defaults:
    metrics_generator:
      processors:
        - span-metrics-latency
        - span-metrics-count
        # span-metrics-size omitted to disable size metrics
```

## How it works

The span metrics processor works by inspecting every received span and computing the total count and the duration of spans for every unique combination of dimensions.
Dimensions can be the service name, the operation, the span kind, the status code and any attribute present in the span.

This processor mirrored the implementation from the OpenTelemetry Collector of the processor with the same name.
The OTel `spanmetricsprocessor` has since been [deprecated](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/processor/spanmetricsprocessor/v0.95.0/processor/spanmetricsprocessor/README.md) and replaced with the [span metric connector](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/processor/spanmetricsprocessor/v0.95.0/connector/spanmetricsconnector/README.md).

{{< admonition type="note" >}}
To learn more about cardinality and how to perform a dry run of the metrics generator, refer to the [Cardinality documentation](ref:cardinality).
{{< /admonition >}}

### Metrics

The following metrics are exported:

| Metric                         | Type      | Labels     | Description                  |
| ------------------------------ | --------- | ---------- | ---------------------------- |
| traces_spanmetrics_latency     | Histogram | Dimensions | Duration of the span         |
| traces_spanmetrics_calls_total | Counter   | Dimensions | Total count of the span      |
| traces_spanmetrics_size_total  | Counter   | Dimensions | Total size of spans ingested |

By default, the metrics processor adds the following labels to each metric: `service`, `span_name`, `span_kind`, and `status_code`.

The `status_message`, `job`, and `instance` labels are optional and require additional configuration, as described in the sections below.

- `service` - The name of the service that generated the span
- `span_name` - The unique name of the span
- `span_kind` - The type of span, this can be one of five values:
  - `SPAN_KIND_SERVER` - The span was generated by a call from another service
  - `SPAN_KIND_CLIENT` - The span made a call to another service
  - `SPAN_KIND_INTERNAL` - The span does not have interaction outside of the service it was generated in
  - `SPAN_KIND_PUBLISHER` - The span created data that was pushed onto a bus or message broker
  - `SPAN_KIND_CONSUMER` - The span consumed data that was on a bus or messaging system
- `status_code` - The result of the span, this can be one of three values:
  - `STATUS_CODE_UNSET` - Result of the span was unset/unknown
  - `STATUS_CODE_OK` - The span operation completed successfully
  - `STATUS_CODE_ERROR` - The span operation completed with an error
- `status_message` (optionally enabled) - The message that details the reason for the `status_code` label
- `job` - The name of the job, a combination of namespace and service; only added if `metrics_generator.processor.span_metrics.enable_target_info: true`
- `instance` - The instance ID; only added if `metrics_generator.processor.span_metrics.enable_target_info: true` and `metrics_generator.processor.span_metrics.enable_instance_label: true`

### Disabling intrinsic dimensions

You can control which intrinsic dimensions are included in your metrics. Disable any of the default intrinsic dimensions using the `intrinsic_dimensions` configuration. This is useful for reducing cardinality when certain labels are not needed.

```yaml
metrics_generator:
  processor:
    span_metrics:
      intrinsic_dimensions:
        service: true
        span_name: true
        span_kind: false # Disable span_kind label
        status_code: true
        status_message: false # Disabled by default
```

### Adding custom dimensions

Additional user defined labels can be created using the [`dimensions` configuration option](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration#metrics-generator).
When a configured dimension collides with one of the default labels (for example, `status_code`), the label for the respective dimension is prefixed with double underscore (for example, `__status_code`).

### Renaming dimensions with dimension_mappings

Custom labeling of dimensions is also supported using the [`dimension_mappings` configuration option](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration#metrics-generator).

**Understanding dimensions vs dimension_mappings:**

Use `dimensions` when you want to add span attributes as labels using their default (sanitized) names. Use `dimension_mappings` when you want to rename attributes to custom label names or combine multiple attributes.

When using `dimension_mappings`, you do not need to also list the same attributes in `dimensions`. The `dimension_mappings` configuration reads directly from the original span attributes.
You can use `dimension_mappings` to rename a single attribute to a different label name, or to combine multiple attributes into a single composite label.

{{< admonition type="note" >}}
The `source_labels` field must contain the **original span or resource attribute names** (with dots), not sanitized Prometheus label names. For example, use `deployment.environment`, not `deployment_environment`.
{{< /admonition >}}

The `name` field can use either dots (`.`) or underscores (`_`), as both are converted to underscores (`_`) in the final Prometheus metric labels. For example, both `env` and `env.label` result in `env_label` in Prometheus metrics.

The following example shows how to rename the `deployment.environment` attribute to a shorter label called `env`, for example:

```yaml
dimension_mappings:
  - name: env
    source_labels: ["deployment.environment"]
```

This example shows how to combine the `service.name`, `service.namespace`, and `service.version` attributes into a single label called `service_instance`. The `join` parameter specifies the separator used to join the attribute values together.

```yaml
dimension_mappings:
  - name: service_instance
    source_labels: ["service.name", "service.namespace", "service.version"]
    join: "/"
```

With this configuration, if a span has the following attribute values:

- `service.name = "abc"`
- `service.namespace = "def"`
- `service.version = "ghi"`

The resulting metric label is `service_instance="abc/def/ghi"`.

An optional metric called `traces_target_info` using all resource level attributes as dimensions can be enabled in the [`enable_target_info` configuration option](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration#metrics-generator).

### Excluding dimensions from target_info

When `enable_target_info` is enabled, all resource attributes are included as labels on the `traces_target_info` metric. To reduce cardinality, you can exclude specific attributes using the `target_info_excluded_dimensions` configuration:

```yaml
metrics_generator:
  processor:
    span_metrics:
      enable_target_info: true
      target_info_excluded_dimensions:
        - "telemetry.sdk.version"
        - "process.runtime.version"
```

### Handling sampled traces

If you use a ratio-based sampler, you can use the custom sampler below to not lose metric information. However, you also need to set `metrics_generator.processor.span_metrics.span_multiplier_key` to `"X-SampleRatio"`.

```go
package tracer
import (
	"go.opentelemetry.io/otel/attribute"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
)

type RatioBasedSampler struct {
	innerSampler        tracesdk.Sampler
	sampleRateAttribute attribute.KeyValue
}

func NewRatioBasedSampler(fraction float64) RatioBasedSampler {
	innerSampler := tracesdk.TraceIDRatioBased(fraction)
	return RatioBasedSampler{
		innerSampler:        innerSampler,
		sampleRateAttribute: attribute.Float64("X-SampleRatio", fraction),
	}
}

func (ds RatioBasedSampler) ShouldSample(parameters tracesdk.SamplingParameters) tracesdk.SamplingResult {
	sampler := ds.innerSampler
	result := sampler.ShouldSample(parameters)
	if result.Decision == tracesdk.RecordAndSample {
		result.Attributes = append(result.Attributes, ds.sampleRateAttribute)
	}
	return result
}

func (ds RatioBasedSampler) Description() string {
	return "Ratio Based Sampler which gives information about sampling ratio"
}
```

### Filtering

In some cases, you may want to reduce the number of metrics produced by the `spanmetrics` processor.
To do so you can configure any of the following processors, in any order or combination:

- `include`: Defines a matching criteria that all spans must meet. If multiple include policies are defined, a span must match all of them to be included (logical AND).

- `include_any`: If a span matches any include_any policy, it is immediately included, bypassing the stricter `include` requirements (logical OR). This is ideal for capturing specific internal spans without opening the floodgates for all internal telemetry.

- `exclude`: If a span matches any exclude policy, it is rejected, even if it matched an inclusion rule.

Currently, only filtering by resource and span attributes with the following value types is supported.

- `bool`
- `double`
- `int`
- `string`

Additionally, these intrinsic span attributes may be filtered upon:

- `name`
- `status` (code)
- `kind`

The following intrinsic kinds are available for filtering.

- `SPAN_KIND_SERVER`
- `SPAN_KIND_INTERNAL`
- `SPAN_KIND_CLIENT`
- `SPAN_KIND_PRODUCER`
- `SPAN_KIND_CONSUMER`

Intrinsic keys can be acted on directly when implementing a filter policy. For example:

```yaml
---
metrics_generator:
  processor:
    span_metrics:
      filter_policies:
        - include:
            match_type: strict
            attributes:
              - key: kind
                value: SPAN_KIND_SERVER
```

In this example, spans which are of `kind` "server" are included for metrics export.

When selecting spans based on non-intrinsic attributes, it is required to specify the scope of the attribute, similar to how it is specified in TraceQL.
For example, if the `resource` contains a `location` attribute which is to be used in a filter policy, then the reference needs to be specified as `resource.location`.
This requires users to know and specify which scope an attribute is to be found and avoids the ambiguity of conflicting values at differing scopes. The following may help illustrate.

```yaml
---
metrics_generator:
  processor:
    span_metrics:
      filter_policies:
        - include:
            match_type: strict
            attributes:
              - key: resource.location
                value: earth
```

In the above examples, we are using `match_type` of `strict`, which is a direct comparison of values.
You can use `regex`, an additional option for `match_type`, to build a regular expression to match against.

```yaml
---
metrics_generator:
  processor:
    span_metrics:
      filter_policies:
        - include:
            match_type: regex
            attributes:
              - key: resource.location
                value: eu-.*
        - exclude:
            match_type: regex
            attributes:
              - key: resource.tier
                value: dev-.*
```

In the above, we first include all spans which have a `resource.location` that begins with `eu-` with the `include` statement, and then exclude those with begin with `dev-`.
In this way, a flexible approach to filtering can be achieved to ensure that only metrics which are important are generated.

```yaml
---
metrics_generator:
  processor:
    span_metrics:
      filter_policies:
        # Only process spans from EU production environments
        - include:
            match_type: regex
            attributes:
              - key: resource.location
                value: eu-.*
        # Exception Rule: Allow INTERNAL spans for auth-service specifically
        - include_any:
            match_type: strict
            attributes:
              - key: kind
                value: SPAN_KIND_INTERNAL
              - key: resource.service.name
                value: auth-service
        # Drop any spans from development tiers
        - exclude:
            match_type: regex
            attributes:
              - key: resource.tier
                value: dev-.*
```

In the above, we want to capture metrics for all production spans in the EU, but we also want to explicitly allow INTERNAL spans from the auth-service, which would otherwise be ignored by the include filter.

## Example

<p align="center"><img src="/media/docs/tempo/metrics/span-metrics-example.png" alt="Span metrics overview"></p>
