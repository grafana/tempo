---
title: Tail-based sampling
menuTItle: Tail-based sampling
description: Use tail-based sampling to optimize sampling decisions
weight:
aliases:
- /docs/tempo/grafana-agent/tail-based-sampling
---

# Tail-based sampling

Tempo aims to provide an inexpensive solution that makes 100% sampling possible.
However, sometimes constraints make a lower sampling percentage necessary or desirable,
such as runtime or egress traffic related costs.
Probabilistic sampling strategies are easy to implement,
but also run the risk of discarding relevant data that you'll later want.

Tail-based sampling works with Grafana Agent in Flow or static modes.
Flow mode configuration files are [written in River](https://grafana.com/docs/agent/latest/flow/config-language).
Static mode configuration files are [written in YAML](https://grafana.com/docs/agent/latest/static/configuration).
Examples in this document are for Flow mode. You can also use the [Static mode Kubernetes operator](https://grafana.com/docs/agent/latest/operator).

## How tail-based sampling works

In tail-based sampling, sampling decisions are made at the end of the workflow allowing for a more accurate sampling decision.
The Grafana Agent groups spans by trace ID and checks its data to see
if it meets one of the defined policies (for example, `latency` or `status_code`).
For instance, a policy can check if a trace contains an error or if it took
longer than a certain duration.

A trace is sampled if it meets at least one policy.

To group spans by trace ID, the Agent buffers spans for a configurable amount of time,
after which it considers the trace complete.
Longer running traces are split into more than one.
However, waiting longer times increases the memory overhead of buffering.

One particular challenge of grouping trace data is for multi-instance Agent deployments,
where spans that belong to the same trace can arrive to different Agents.
To solve that, you can configure the Agent to load balance traces across agent instances
by exporting spans belonging to the same trace to the same instance.

This is achieved by redistributing spans by trace ID once they arrive from the application.
The Agent must be able to discover and connect to other Agent instances where spans for the same trace can arrive.
Kubernetes users should use a [headless service](https://kubernetes.io/docs/concepts/services-networking/service/#headless-services).

Redistributing spans by trace ID means that spans are sent and received twice,
which can cause a significant increase in CPU usage.
This overhead increases with the number of Agent instances that share the same traces.

<p align="center"><img src="../tail-based-sampling.png" alt="Tail-based sampling overview"></p>

## Quickstart

To start using tail-based sampling, define a sampling policy.
If you're using a multi-instance deployment of the agent,
add load balancing and specify the resolving mechanism to find other Agents in the setup.
To see all the available configuration options, refer to the [configuration reference](/docs/agent/latest/configuration/traces-config/).

## Example for Grafana Agent Flow

[Grafana Agent Flow](https://grafana.com/docs/agent/latest/flow/) is a component-based revision of Grafana Agent with a focus on ease-of-use, debuggability, and ability to adapt to the needs of power users.
Flow configuration files written in River instead of YAML.

Grafana Agent Flow uses the [`otelcol.processor.tail_sampling component`](https://grafana.com/docs/agent/latest/flow/reference/components/otelcol.processor.tail_sampling/)` for tail-based sampling.

```river
otelcol.receiver.otlp "otlp_receiver" {
    grpc {
        endpoint = "0.0.0.0:4317"
    }

    output {
        traces = [
            otelcol.processor.tail_sampling.errors.input,
        ]
    }
}

otelcol.exporter.otlp "tempo" {
    client {
        endpoint = "tempo:4317"
    }
}

// The Tail Sampling processor will use a set of policies to determine which received traces to keep
// and send to Tempo.
otelcol.processor.tail_sampling "errors" {
    // Total wait time from the start of a trace before making a sampling decision. Note that smaller time
    // periods can potentially cause a decision to be made before the end of a trace has occurred.
    decision_wait = "30s"
    // The number of traces to keep in memory by the Agent.
    num_traces = 100

    // The following policies follow a logical OR pattern, meaning that if any of the policies match,
    // the trace will be kept. For logical AND, you can use the `and` policy. Every span of a trace is
    // examined by each policy in turn. A match will cause a short-circuit.

    // This policy defines that traces that contain errors should be kept.
    policy {
        // The name of the policy can be used for logging purposes.
        name = "only-sample-erroring-traces"
        // The type must match the type of policy to be used, in this case examing the status code
        // of every span in the trace.
        type = "status_code"
        // This block determines the error codes that should match in order to keep the trace,
        // in this case the OpenTelemetry 'ERROR' code.
        status_code {
            status_codes = [ "ERROR" ]
        }
    }

    // This policy defines that only traces that are longer than 200ms in total should be kept.
    policy {
        // The name of the policy can be used for logging purposes.
        name = "only-sample-long-traces"
        // The type must match the policy to be used, in this case the total latency of the trace.
        type = "latency"
        // This block determines the total length of the trace in milliseconds.
        latency {
            threshold_ms = 200
        }
    }

    // The output block forwards the kept traces onto the batch processor, which will marshall them
    // for exporting to Tempo.
    output {
        traces = [otelcol.processor.batch.default.input]
    }
}

```

## Examples for Grafana Agent static mode

For additional information, refer to the blog post, [An introduction to trace sampling with Grafana Tempo and Grafana Agent](https://grafana.com/blog/2022/05/11/an-introduction-to-trace-sampling-with-grafana-tempo-and-grafana-agent/).

### Status code tail sampling policy

The following policy only samples traces where at least one span contains an OpenTelemetry Error status code.

```yaml
traces:
  configs:
    - name: default
    ...
    tail_sampling:
      policies:
        - type: status_code
          status_code:
            status_codes:
              - ERROR
```

### Span attribute tail sampling policy

The following policy will only sample  traces where the span attribute `http.target` does *not* contain the value `/healthcheck` or is prefixed with `/metrics/`.

```yaml
traces:
   configs:
will only sample  traces where the span attribute `http.target` does *not* contain the value `/healthcheck` or is prefixed with `/metrics/`.
   - name: default
    tail_sampling:
      policies:
        - type: string_attribute
          string_attribute:
            key: http.target
            values:
              - ^\/(?:metrics\/.*|healthcheck)$
            enabled_regex_matching: true
            invert_match: true
```
``````` tail_sampling:
      policies:

### And compound tail sampling policy
The following policy will only sample traces where all of the conditions for the sub-policies are met. In this case, it takes the prior two policies and will only sample traces where the span attribute `http.target` does *not* contain the value `/healthcheck` or is prefixed with `/metrics/` *and* at least one of the spans of the trace contains an OpenTelemetry Error status code.
        - type: and
   yaml       and:
            and_sub_policy:
            - name: and_tag_policy
              type: string_attribute
              string_attribute:
                key: http.target
                values:
                    - ^\/(?:metrics\/.*|healthcheck)$
                enabled_regex_matching: true
                invert_match: true
            - name: and_error_policy
              type: status_code
              status_code:
                status_codes:
                  - ERROR