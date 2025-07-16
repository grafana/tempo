---
title: Enable tail-based sampling
menuTitle: Enable tail sampling
description: Configure tail sampling with Tempo and Grafana Alloy to optimize sampling decisions.
---

# Enable tail sampling

You can use tail sampling to use a lower sampling percentage when necessary or desirable,
such as runtime or egress traffic related costs.
Probabilistic sampling strategies are easy to implement,
but also run the risk of discarding relevant data that you'll later want.

Tail sampling works with Grafana Alloy.
Alloy configuration files are written in [Alloy configuration syntax](https://grafana.com/docs/alloy/<ALLOY_VERSION>/get-started/configuration-syntax/).

## Configure tail sampling

To start using tail sampling, define a sampling policy in your configuration file.

If you're using a multi-instance deployment of Alloy,
add load balancing and specify the resolving mechanism to find other Alloy instances in the setup.

To see all the available configuration options for load balancing, refer to the [Alloy component reference](https://grafana.com/docs/alloy/<ALLOY_VERSION>/reference/components/otelcol.exporter.loadbalancing/).

### Example for Alloy

Alloy uses the [`otelcol.processor.tail_sampling component`](https://grafana.com/docs/alloy/<ALLOY_VERSION>/reference/components/otelcol/otelcol.processor.tail_sampling/) for tail sampling.

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
