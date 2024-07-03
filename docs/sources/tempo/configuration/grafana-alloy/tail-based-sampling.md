---
title: Enable tail-based sampling
menuTItle: Enable tail-based sampling
description: Use tail-based sampling to optimize sampling decisions
weight:
aliases:
- /docs/tempo/grafana-alloy/tail-based-sampling
---

# Enable tail-based sampling

Tempo provides an inexpensive solution that aims to reduce the amount of tail-based sampling required.
However, sometimes constraints make a lower sampling percentage necessary or desirable,
such as runtime or egress traffic related costs.
Probabilistic sampling strategies are easy to implement,
but also run the risk of discarding relevant data that you'll later want.

Tail-based sampling works with Grafana Alloy.
Alloy configuration files are written in [Alloy configuration syntax](https://grafana.com/docs/alloy/latest/concepts/configuration-syntax/).

## How tail-based sampling works

In tail-based sampling, sampling decisions are made at the end of the workflow allowing for a more accurate sampling decision.
Alloy groups spans by trace ID and checks its data to see
if it meets one of the defined policies (for example, `latency` or `status_code`).
For instance, a policy can check if a trace contains an error or if it took
longer than a certain duration.

A trace is sampled if it meets at least one policy.

To group spans by trace ID, Alloy buffers spans for a configurable amount of time,
after which it considers the trace complete.
Longer running traces are split into more than one.
However, waiting longer times increases the memory overhead of buffering.

One particular challenge of grouping trace data is for multi-instance Alloy deployments,
where spans that belong to the same trace can arrive to different Alloys.
To solve that, you can configure Alloy to load balance traces across Alloy instances
by exporting spans belonging to the same trace to the same instance.

This is achieved by redistributing spans by trace ID once they arrive from the application.
Alloy must be able to discover and connect to other Alloy instances where spans for the same trace can arrive.
Kubernetes users should use a [headless service](https://kubernetes.io/docs/concepts/services-networking/service/#headless-services).

Redistributing spans by trace ID means that spans are sent and received twice,
which can cause a significant increase in CPU usage.
This overhead increases with the number of Alloy instances that share the same traces.

<p align="center"><img src="../tempo-tail-based-sampling.svg" alt="Tail-based sampling overview"></p>

## Configure tail-based sampling

To start using tail-based sampling, define a sampling policy in your configuration file.

If you're using a multi-instance deployment of Alloy,
add load balancing and specify the resolving mechanism to find other Alloy instances in the setup.

To see all the available configuration options for load balancing, refer to the [Alloy component reference](https://grafana.com/docs/alloy/latest/reference/components/otelcol.exporter.loadbalancing/).

### Example for Alloy

Alloy uses the [`otelcol.processor.tail_sampling component`](https://grafana.com/docs/alloy/latest/reference/components/otelcol.processor.tail_sampling/) for tail-based sampling.

```alloy
otelcol.receiver.otlp "default" {
  http {}
  grpc {}

  output {
    traces  = [otelcol.processor.tail_sampling.policies.input]
  }
}

// The Tail Sampling processor will use a set of policies to determine which received
// traces to keep and send to Tempo.
otelcol.processor.tail_sampling "policies" {
    // Total wait time from the start of a trace before making a sampling decision.
    // Note that smaller time periods can potentially cause a decision to be made
    // before the end of a trace has occurred.
    decision_wait = "30s"

    // The following policies follow a logical OR pattern, meaning that if any of the
    // policies match, the trace will be kept. For logical AND, you can use the `and`
    // policy. Every span of a trace is examined by each policy in turn. A match will
    // cause a short-circuit.

    // This policy defines that traces that contain errors should be kept.
    policy {
        // The name of the policy can be used for logging purposes.
        name = "sample-erroring-traces"
        // The type must match the type of policy to be used, in this case examining
        // the status code of every span in the trace.
        type = "status_code"
        // This block determines the error codes that should match in order to keep
        // the trace, in this case the OpenTelemetry 'ERROR' code.
        status_code {
            status_codes = [ "ERROR" ]
        }
    }

    // This policy defines that only traces that are longer than 200ms in total
    // should be kept.
    policy {
        // The name of the policy can be used for logging purposes.
        name = "sample-long-traces"
        // The type must match the policy to be used, in this case the total latency
        // of the trace.
        type = "latency"
        // This block determines the total length of the trace in milliseconds.
        latency {
            threshold_ms = 200
        }
    }

    // The output block forwards the kept traces onto the batch processor, which
    // will marshall them for exporting to the Grafana OTLP gateway.
    output {
        traces = [otelcol.exporter.otlp.default.input]
    }
}

otelcol.exporter.otlp "default" {
  client {
    endpoint = env("OTLP_ENDPOINT")
  }
}
```
