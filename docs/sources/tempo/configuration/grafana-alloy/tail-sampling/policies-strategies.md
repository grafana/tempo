---
title: Tail sampling policies and strategies
menuTitle: Tail sampling policies and strategies
description: Learn about tail sampling policies and strategies in Tempo and Grafana Alloy.
weight: 700
---

# Tail sampling policies and strategies

[Tail sampling strategies](https://grafana.com/docs/tempo/latest/configuration/grafana-alloy/tail-sampling/) consider all, or a subset, of the spans that have been collected by an OpenTelemetry Collector [distribution](https://opentelemetry.io/docs/concepts/distributions/), such as [Grafana Alloy](https://grafana.com/docs/alloy/<ALLOY_VERSION>/).

Tail sampling is currently defined as part of a telemetry pipeline.
Alloy and other collectors are part of the `processing` set of components that are executed after telemetry has been received by Alloy, but before it is exported to a trace storage system such as to Grafana Tempo or Grafana Cloud Traces.

In the context of OpenTelemetry, tail sampling is implemented by configuring sampling policies.
A sampling policy provides the criteria that makes a decision to sample or discard a trace.
These criteria might include decisions based on specific response status codes, trace duration, attribute values, or other custom-defined rules.
Tail sampling operates as part of the telemetry processing pipeline in Alloy: it can make informed decisions based on the entire trace as opposed to isolated spans.
The sampling criteria policies are defined within a tail sampling block. 
For more information, refer to [tail sampling processor block](https://grafana.com/docs/alloy/<ALLOY_VERSION>/reference/components/otelcol/otelcol.processor.tail_sampling/).

For example, this basic Alloy configuration receives OTLP data either via HTTP or gRPC and passes incoming trace spans into the tail sampling processor.
Before sending the trace spans to a tracing store like Tempo, the tail sampling processor decides whether or not to sample the trace based on the probabilistic policy. 
The probabilistic policy simply samples a specified percentage of traces observed randomly.
Defaults are used for the tail sampling processor.

```alloy
// Expose receiving OTLP data
otelcol.receiver.otlp "example" {
  // Allow OTLP HTTP data on all interfaces on port 4318
  http {
    endpoint = "0.0.0.0:4318"
  }
  // Allow OTLP gRPC data on all interfaces on port 4317
  grpc {
    endpoint = "0.0.0.0:4317"
  }

  // Send all received trace spans to the tail sampling processor
  output {
    traces = [ otelcol.processor.tail_sampling.example.input ]
  }
}

// Make decisions on whether to sample or discard traces
otelcol.processor.tail_sampling "example" {
  // We define a single probabilistic processor to determine sampling
  policy {
    // The name of the policy, each policy name must be unique for this tail sampling instance
    name = "example_probabilistic"
    // The policy type is probabilistic
    type = "probabilistic"

    // Each policy type is defined by a block for the policy with specific parameters for it
    probabilistic {
      // The overall ratio of traces that have been received to randomly sample.
      // In this case 1 in 10.
      sampling_percentage = 10
    }
  }

  // Output all sampled trace spans to the OTLP exporter.
  output {
    traces = [ otelcol.exporter.otlp.example.input ]
  }
}

// The OTLP exporter sends telemetry onwards to a downstream destination.
otelcol.exporter.otlp "example" {
  // The client block defines the target destination.
  client {
    // The endpoint denotes the location of the host receiving the sampled trace
    // data. In this case a local Tempo instance.
    endpoint = "http://tempo:4318"
  }
}
```

## Policy types

The table lists available policy types for the tail sampling processor. 
For additional information, refer to [`otelcol.processor.tail_sampling` component](https://grafana.com/docs/alloy/<ALLOY_VERSION>/reference/components/otelcol/otelcol.processor.tail_sampling/)` in the Alloy documentation and the [Tail Sampling Processor README](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/processor/tailsamplingprocessor/README.md) for the OTel Collector.

| Policy                                                                                                                                           | Description                                                                                                    | Useful for                                                              |
| ------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| always_sample                                                                                                                                    | Samples all traces.                                                                                            | Debugging or collecting all data.                                       |
| [and](https://grafana.com/docs/alloy/<ALLOY_VERSION>/reference/components/otelcol/otelcol.processor.tail_sampling/#and-block)                             | Lets you combine multiple policies using a logical `AND` operation.   | Activating one or more policies.                                        |
| [boolean_attribute](https://grafana.com/docs/alloy/<ALLOY_VERSION>/reference/components/otelcol/otelcol.processor.tail_sampling/#boolean_attribute-block) | Samples based on a boolean attribute (resource and record).                                                    | Feature flags or debug modes.                                           |
| [composite](https://grafana.com/docs/alloy/<ALLOY_VERSION>/reference/components/otelcol/otelcol.processor.tail_sampling/#composite-block)                 | Samples based on a combination of samplers, with ordering and rate allocation per sampler.                     | Matching multiple different conditions.                                 |
| [latency](https://grafana.com/docs/alloy/<ALLOY_VERSION>/reference/components/otelcol/otelcol.processor.tail_sampling/#latency-block)                     | Samples traces based on their duration.                                                                        | Identifying slow performance.                                           |
| [numeric_attribute](https://grafana.com/docs/alloy/<ALLOY_VERSION>/reference/components/otelcol/otelcol.processor.tail_sampling/#numeric_attribute-block) | Samples based on the number attributes (resource and record).                                                  | Capturing large responses.                                              |
| [ottl_condition](https://grafana.com/docs/alloy/<ALLOY_VERSION>/reference/components/otelcol/otelcol.processor.tail_sampling/#ottl_condition-block)       | Samples based on a given boolean OpenTelemetry Transformation Language (OTTL) condition (span and span event). | Applying complex and specific filtering.                                |
| [probabilistic](https://grafana.com/docs/alloy/<ALLOY_VERSION>/reference/components/otelcol/otelcol.processor.tail_sampling/#probabilistic-block)         | Samples a percentage of traces.                                                                                | Filtering only a percentage of received traces. Reducing data received. |
| [rate_limiting](https://grafana.com/docs/alloy/<ALLOY_VERSION>/reference/components/otelcol/otelcol.processor.tail_sampling/#rate_limiting-block)         | Samples based on rate of spans per second.                                                                     | Controlling data volume.                                                |
| [span_count](https://grafana.com/docs/alloy/<ALLOY_VERSION>/reference/components/otelcol/otelcol.processor.tail_sampling/#span_count-block)               | Samples based on the minimum number of spans within the observed trace.                                        | Limiting sampled data to a specific number of spans within a trace.     |
| [status_code](https://grafana.com/docs/alloy/<ALLOY_VERSION>/reference/components/otelcol/otelcol.processor.tail_sampling/#status_code-block)             | Samples based upon the status code, either OK, Error, or Unset.                                                | Capturing erroring traces.                                              |
| [string_attribute](https://grafana.com/docs/alloy/<ALLOY_VERSION>/reference/components/otelcol/otelcol.processor.tail_sampling/#string_attribute-block)   | Samples based on string attributes (resource and record) value matches.                                        | Filtering specific services or database queries.                        |
| [trace_state](https://grafana.com/docs/alloy/<ALLOY_VERSION>/reference/components/otelcol/otelcol.processor.tail_sampling/#trace_state-block)             | Samples based on TraceState value matches. |


## Sampling policies and use cases

This section provides examples for the policy strategies.

{{< admonition type = "note" >}}
Sample data at the collector after metrics generation so that all traces are available to generate accurate metrics. If you generate metrics from sampled traces, the sampling affects their values.
{{< /admonition >}}

### Always sample

You can use `always_sample` when you want to capture all tracing data. This could be useful for troubleshooting.

Refer to the <code>[always_sample](https://grafana.com/docs/<ALLOY_VERSION>/latest/reference/components/otelcol/otelcol.processor.tail_sampling/#always_sample)</code> documentation for more information.` `

```alloy
policy {
    // The example below is for the `always_sample` policy
    name = "example_always_sample"
    type = "always_sample"
}
```

### And

You use the `and` sampling policy when you want to match on multiple conditions. This example uses a probabilistic sampler and the latency sampling policy.

You can use this to look for slow requests and sample a percentage of traces.

Refer to the [`and`](https://grafana.com/docs/alloy/<ALLOY_VERSION>/reference/components/otelcol/otelcol.processor.tail_sampling/#and) documentation for more information.

```alloy
policy {
    // The example below is for the `and` sampling policy
    name = "example_and"
    type = "and"
    and {
      policies = [
        name = "example_probabilistic"
        type = "probabilistic"
        probabilistic {
         // The percentage of traces to "randomly" sample.
         sampling_percentage = 15
        },
        name = "example_latency"
        type = "latency"
        latency {
         // The minimum duration for a trace to be sampled
         threshold_ms = 5000
       }
    ]
  }
}
```

### Boolean attribute

Use [`boolean_attribute`](https://grafana.com/docs/alloy/<ALLOY_VERSION>/reference/components/otelcol/otelcol.processor.tail_sampling/#boolean_attribute-block) to sample based on whether a specific span attribute with a boolean value is true or false. Any span with the named span attribute set to the given boolean value will cause the trace to be sampled.

Refer to the [`boolean_attribute`](https://grafana.com/docs/alloy/<ALLOY_VERSION>/reference/components/otelcol/otelcol.processor.tail_sampling/#boolean_attribute) documentation for more information.

```alloy
policy {
    // The example below is for the `boolean_attribute` sampling policy
    name = "example_boolean_attribute"
    type = "boolean_attribute"

    boolean_attribute {
      // The span or resource attribute to be considered.
      key = "my.boolean"
      // Sample the trace if the value is boolean and set to `true`.
      value = true
    }
  }
```

### Composite

This policy, similar to `and`, is built up of multiple sub-policies.
Unlike `and`, `composite` specifies a maximum throughput of spans to be sampled each second.
Each sub-policy is given a weighting by percentage of maximum throughput.
Should the percentage be maxed out during evaluation, then the other policies in the given order are evaluated instead.
Because `composite` is evaluated just like any other sampling policy, it can be used in conjunction with others to act as final decision maker to limit trace sample output should no other policies match.
Generally, composite policies end with an `always_sample` policy type to ensure that traces are still sampled should none of the other aggregated policies inside it match.

Refer to the [`composite`](https://grafana.com/docs/alloy/<ALLOY_VERSION>/reference/components/otelcol/otelcol.processor.tail_sampling/#composite-block) documentation for more information.

```alloy
policy {
    // The example below is for the `composite` sampling policy
    name = "composite-policy"
    type = "composite"
    composite {
            // Limit sampling to a maximum of 500 spans per second.
            max_total_spans_per_second = 500
            // Evaluate the policies in the following order.
            // This acts like the default drop-through policy matcher.
            policy_order = ["composite-policy-keyvalue", "composite-policy-always"]
            // Sample any trace with a 20x status code.
            composite_sub_policy {
                name = "composite-policy-keyvalue"
                type = "string_attribute"
                string_attribute {
                    key = "http.code"
                    enabled_regex_matching = true
                    values = [ "20[0|1|2]" ]
                }
            }
            // Finalise with an `always_sample` policy type.
            composite_sub_policy {
                name = "composite-policy-always"
                type = "always_sample"
            }
            // Allocate 80% of sampling to the `string_attribute` policy.
            // The remaining 20% is allocated to the `always_sample` policy.
            rate_allocation {
                policy = "composite-policy-keyvalue"
                percent = 80
            }
            rate_allocation {
                policy = "composite-policy-always"
                percent = 20
            }
        }
    }
```

### Latency

The latency policy is used to look for slow-running spans. 
For example, you can use it for performance monitoring. 
Slow-running spans can indicate performance bottlenecks.
The example samples for traces that are between 3s and 10s long.

Refer to the [`latency`](https://grafana.com/docs/alloy/<ALLOY_VERSION>/reference/components/otelcol/otelcol.processor.tail_sampling/#latency) documentation for more information.

```alloy
policy {
    // The example below is for the `latency` sampling policy
    name = "example_latency"
    type = "latency"

    latency {
      // The minimum duration for a trace to be sampled
      threshold_ms = 3000
      // The maximum duration for a trace to be sampled
      max_duration_ms = 10000
    }
  }
```

### Numeric attribute

The `numeric_attribute` policy lets you sample based on a number of attributes. 
In this example below, spans are sampled where `http.response_content_length` is between 10,000 and 500,000. 
The sample span below would be captured because the attribute value falls within the specified range.

Refer to the [`numeric_attribute`](https://grafana.com/docs/alloy/<ALLOY_VERSION>/reference/components/otelcol/otelcol.processor.tail_sampling/#numeric_attribute) documentation for more information.

```alloy
policy {
    // The example below is for the `numeric_attribute` sampling policy
    name = "example_numeric_attribute"
    type = "numeric_attribute"

    numeric_attribute {
      // The span or resource attribute to be considered.
      key = "http.response_content_length"
      // The minimum value for the attribute to be sampled.
      min_value = 10000
      // The maximum value for the attribute to be sampled.
      max_value = 500000
    }
  }
```

### OTTL condition

This policy type lets you configure policies based around OTTL. The various language semantics let you write flexible and detailed sampling policies that can evaluate any trace span conditions to determine whether or not to sample a trace.
If OTTL conditions are particularly complex, this flexibility can potentially create resource overhead.

Refer to the [`ottl_condition`](https://grafana.com/docs/alloy/<ALLOY_VERSION>/reference/components/otelcol/otelcol.processor.tail_sampling/#ottl_condition) documentation for more information.

```alloy
policy {
    // The example below is for the `ottl_condition` sampling policy
    name = "example_ottl_condition"
    type = "ottl_condition"

    ottl_condition {
        condition = "resource.attribute['service.name'] == 'checkout_service'"
    }
  }
```

### Probabilistic

Probabilistic sampling lets you determine a ratio of traces that should be sampled.
This is within a range of `0` to `100%`, and selects the required percentage of traces from those it receives based on the number of traces per second configured.
This can be combined with varying hashing salt values, should multiple collectors be carrying out sampling simultaneously, for example, in a load balanced sampling hierarchy).

Refer to the [`probabilistic`](https://grafana.com/docs/alloy/<ALLOY_VERSION>/reference/components/otelcol/otelcol.processor.tail_sampling/#probabilistic-block) documentation for more information.

```alloy
policy {
    // The example below is for the `probabilistic` sampling policy
    name = "example_probabilistic"
    type = "probabilistic"

    probabilistic {
	// The percentage of traces to "randomly" sample.
      sampling_percentage = 10
    }
  }
```

### Rate limiting

This policy type samples traces based around the configured spans per second.
This ensures that only traces that conform to the number of those spans available in that trace are sampled, any trace with greater than this number of spans will automatically be dropped.

Refer to the [`rate_limiting`](https://grafana.com/docs/alloy/<ALLOY_VERSION>/reference/components/otelcol/otelcol.processor.tail_sampling/#rate_limiting`) documentation for more information.

```alloy
policy {
    // The example below is for the `rate_limiting` sampling policy
    name = "example_rate_limiting"
    type = "rate_limiting"

    rate_limiting {
       // Defines the maximum number of collective spans in a second that the trace should have for it to be sampled.
	spans_per_second = 20
    }
  }
```

### Span count

The `span_count` policy accepts a window of minimum and maximum number of spans for a trace to be sampled once the decision period is reached.

Setting the minimum number of spans to zero essentially replicates the rate limiting policy.

Refer to the [`span_count`](https://grafana.com/docs/alloy/<ALLOY_VERSION>/reference/components/otelcol/otelcol.processor.tail_sampling/#span_count) documentation for more information.

```alloy
policy {
    // The example below is for the `span_count` sampling policy
    name = "example_span_count"
    type = "span_count"

    span_count {
	min_spans = 5
	max_spans = 50
    }
  }
```

### Status code

The status code policy check for the span intrinsic status of `OK`, `UNSET`, or `ERROR`. The policy takes an array of statuses, so only traces with at least one span that matches these statuses are sampled.

Refer to the [`status_code`](https://grafana.com/docs/alloy/<ALLOY_VERSION>/reference/components/otelcol/otelcol.processor.tail_sampling/#status_code) documentation for more information.

```alloy
policy {
    // The example below is for the `status_code` sampling policy
    name = "example_status_code"
    type = "status_code"

    status_code {
      // An array of status codes; should any of the spans in a trace match one
      // of the status codes, it will be sampled.
      status_codes = ["ERROR"]
    }
  }
```

### String attribute

This policy examines the values of a specified string attribute key for trace’s spans, to make a sampling decision.
There are a number of configurable options, including the ability to use regular expressions, inversion matching, for example, to only sample traces that don't match a regular expression, and an LRU cache for accelerating future policy decision making.

Refer to the [`string_attribute`](https://grafana.com/docs/alloy/<ALLOY_VERSION>/reference/components/otelcol/otelcol.processor.tail_sampling/#string_attribute) documentation for more information.

```alloy
policy {
    // The example below is for the `string_attribute` sampling policy
    name = "example_string_attribute"
    type = "string_attribute"

    string_attribute {
      // The span or resource attribute to be considered.
      key = "db.statement"
      // Enables regular expression matching to be defined in the attribute values defined in `values`. Defaults to false for exact pattern matching.
      enabled_regex_matching = true
      // The number of LRU cache entries to save, to allow less full regexps to be run during decision making.
      cache_max_size = 100
      // Determines if a value match should determine whether the trace is sampled. `false` to only sample traces when the attribute's value matches, `true` to only sample traces where the attribute's value does not match.
      invert_match = false
      values = ["SELECT.*", "INSERT.*"]
    }
  }
```

### Trace state

The `trace_state` policy looks at the traces states associated with the incoming spans for traces and then compares the keys, should any exist, with those configured by the policy. This ensures that only traces with the relevant trace state keys are sampled.

Refer to the [`trace_state`](https://grafana.com/docs/alloy/<ALLOY_VERSION>/reference/components/otelcol/otelcol.processor.tail_sampling/#trace_state) documentation for more information.

```alloy
policy {
    // The example below is for the `trace_state` sampling policy
    name = "example_trace_state"
    type = "trace_state"

    trace_state {
	key = "sampling.priority"
	values = ["1"]
    }
  }
```
