---
title: Sampling
menuTItle: Sampling
description: Use sampling to optimize sampling decisions
weight:
aliases:
- /docs/tempo/grafana-alloy/tail-based-sampling
- ../tail-based-sampling/
---

# Sampling

Grafana Tempo provides an inexpensive solution that aims to ingest and store the traces that provide maximum observability across your application estate.
However, sometimes constraints mean that storing all of your traces is not desirable, for example runtime or egress traffic related costs.
There are a number of ways to lower trace volume, including varying sampling strategies.

Sampling is the process of determining which traces to store (in Tempo or Grafana Cloud Traces) and which to discard. Sampling comes in two different strategy types: head and tail sampling.

Sampling functionality exists in both [Grafana Alloy](https://grafana.com/docs/alloy/) and the OpenTelemetry Collector. Alloy can collect, process, and export telemetry signals, with configuration files written in [Alloy configuration syntax](https://grafana.com/docs/alloy/<ALLOY_VERSION>/concepts/configuration-syntax/).

Refer to [Enable tail sampling](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/grafana-alloy/enable-tail-sampling/) for instructions on how to enable tail sampling.

## Head and tail sampling

When sampling, you can use a head or tail sampling strategy.

With a head sampling strategy, the decision to sample the trace is usually made as early as possible and doesn’t need to take into account the whole trace.
It’s a simple but effective sampling strategy.
Refer to the [Head sampling documentation](https://grafana.com/docs/grafana-cloud/monitor-applications/application-observability/collector/sampling/head/#head-sampling) for [Application Observability](https://grafana.com/docs/grafana-cloud/monitor-applications/application-observability/) for more information.

With a tail sampling strategy, the decision to sample a trace is made after considering all or most of the spans. For example, tail sampling is a good option to sample only traces that have errors or traces with long request duration.
Tail sampling is more complex to configure, implement, and maintain but is the recommended sampling strategy for large systems with a high telemetry volume.

For more information about sampling, refer to the [OpenTelemetry Sampling](https://opentelemetry.io/docs/concepts/sampling/) documentation.

![Tail sampling overview and components with Tempo, Alloy, and Grafana](/media/docs/tempo/sampling/tempo-tail-based-sampling.svg)

## How tail sampling works in the OpenTelemetry Tail Sampling Processor

In tail-based sampling, sampling decisions are made at the end of the workflow allowing for a more accurate sampling decision.
Alloy uses the [OpenTelemetry Tail Sampling Processor](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/processor/tailsamplingprocessor/README.md).

Alloy organizes spans by trace ID and evaluates its data to see if it meets one of the defined policy types (for example, `latency` or `status_code`).
For instance, a policy can check if a trace contains an error or the trace duration was longer than a specified threshold.

A trace is sampled if it meets the conditions of at least one policy.

### Decision periods

To group spans by trace ID, Alloy buffers spans for a configurable amount of time, after which it considers the trace complete.
This configurable amount of time is known as the decision period. Longer running traces are split into more than one.

In situations where a specific trace is longer in duration than the decision period, multiple decisions might be made for any future spans that fall outside of the decision period window.
This can result in some spans for a trace being sampled, while others are not.

For example, consider a situation where the tail sampler decision period is 10 seconds, and a single policy exists to sample traces where an error is set on at least one span.
One of the traces is 20 seconds in duration and a single span at time offset 15 seconds exhibits an error status.

![Trace Policy: Error when status exists](/media/docs/tempo/sampling/tempo-decision-point-sampling.svg)

When the first span for the trace is observed, the decision period time of 10 seconds is initiated. Once the decision period has expired, the tail sampler won’t have observed any spans with an error status, and will therefore discard the trace spans.

When the next span for the trace arrives, a new decision period of 10 seconds begins.
In this period, one of the observed spans has an error set on it. When the decision period expires, all of the spans for the trace in that period will be sampled.

This leads to a fragmented trace being stored in Tempo, where only the spans for the last 10 seconds of the trace will be available to query.
While this is still a potentially useful trace, careful determination of how to set the decision period is key to ensuring that trace spans are sampled correctly.

However, using longer decision periods increases the memory overhead of buffering the spans required to make a decision for each trace.

For this reason, enabling a decision cache can ensure that previous sampling decisions for a specific trace ID are honored even after the expiration of the decision period.
For more details, refer to the Caches section.

### Caches

The OpenTelemetry tail sampling processor includes two separate caches, the sampled and non-sampled caches.
The sampled cache keeps a list of all trace IDs where a prior decision to keep spans has been made.
The non-sampled cache keeps a list of all trace IDs where a prior decision to drop spans has been made.
Both caches are configured by the maximum number of traces that should be stored in the cache, and can be enabled either independently or jointly.

![Decision points and caches workflow](/media/docs/tempo/sampling/tempo-alloy-sampling-policies.svg)

In the above diagram, should both caches be enabled, then a decision to drop samples for the trace is made after 10 seconds and the trace ID stored in the non-sampled cache.
This means that even spans that have an error status for that trace are dropped after the initial decision period, as the non-sampled cache matches the trace ID and pre-emptively drops the span.
However, the same is true should a sampled decision have been made, where any future spans do not match any policies but whose trace ID is found in the sampled cache.

Understanding how these caches work ensures that you still keep decisions that have previously been made.
For example, you could use the sampled cache to short-circuit future decisions for a trace, immediately sampling the incoming span.
This allows a decision to be made without having to buffer any other spans.

Here are some general guidelines for using caches.
Every installation is different.
Using the caches can impact the amount of data generated.

| Cache type | Use case | Benefits/Considerations |
|---|---|---|
| Sample caches | Keep any future spans from traces that have been sampled.  | Cuts down span storage per trace to only those matching policies.  <br /> Can cause fragmented tracing. |
| Non-sampled | Drop any future spans from traces where a decision to not sample those traces has explicitly occurred. | Lowers chance of storing traces after the initial decision period. <br/> Will miss any trace whose spans exhibit future policy criteria matching. |
| Both | Use an initial decision period that makes a decision once and uses that decision going forward.  | Guarantees capture of full traces. <br /> Lower chance of capturing useful traces with a long duration. <br /> Can lose spans if they are longer than the decision period. |

{{< admonition type="note" >}}
Enabling both sampled and non-sampled caches exhibits functionality similar to that of not enabling caches.
However, it short-circuits any future decision making once an initial decision period has expired. Enabling both caches lowers memory requirements for buffering spans.
{{< /admonition >}}

### Tail sampling load balancing

Situations may arise for multi-instance Alloy deployments where emitted spans belonging to the same trace may arrive at different instances.
For most cases, sampling decisions rely on all the spans for a specific trace ID being received by a single instance.

You can configure Alloy to load balance traces across instances by exporting spans belonging to a specific trace ID to the same instance.
For example, if 10 traces are coming in and there are four Alloy instances, then each instance will receive three traces and one instance will receive four traces.
The load balancing maintains consistent hashing across all instances.

Tail sampling load balancing is usually carried out by running two layers of collectors.
The first layer receives the telemetry data (in this case trace spans), and then distributes these to the second layer that carries out the sampling policies.

![Load balancing incoming traces using Alloy](/media/docs/tempo/sampling/tempo-alloy-sampling-loadbalancing.svg)

Alloy includes a [load-balancing exporter](https://grafana.com/docs/alloy/<ALLOY_VERSION>/reference/components/otelcol/otelcol.exporter.loadbalancing/) that can carry out routing to further collector targets based on a set number of keys (in the case of trace sampling, usually the `traceID` key).
Alloy uses the [OpenTelemetry load balancing exporter](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/exporter/loadbalancingexporter/README.md).

The routing key ensures that a specific collector in the second layer always handles spans from the same trace ID, guaranteeing that sampling decisions are made correctly.
You can configure the exporter with targets using static IPs, multi-IP DNS A record entries, and a Kubernetes headless service resolver.
Using this configuration lets you scale up or down the number of layer two collectors.

There are some important points to note with the load balancer exporter around scaling and resilience, mostly around its eventual consistency model. For more information, refer to [Resilience and scaling considerations](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/exporter/loadbalancingexporter/README.md#resilience-and-scaling-considerations).
The most important in terms of tail sampling is that routing occurs based on an algorithm taking into account the number of backends available to the load balancer.
This can affect the target for trace ID spans before eventual consistency occurs.

For an example manifest for a two layer OpenTelemetry Collector deployment based around Kubernetes services, refer to the [Kubernetes service resolver README](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/exporter/loadbalancingexporter/example/k8s-resolver/README.md).