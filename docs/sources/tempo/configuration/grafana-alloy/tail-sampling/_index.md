---
title: Sampling
menuTitle: Sampling
description: Learn how to streamline tracing data by using sampling to determine which data to keep and which to drop.
aliases:
- ./tail-based-sampling/ # /docs/tempo/latest/configuration/grafana-alloy/tail-based-sampling/
---

# Sampling

Grafana Tempo is a cost-effective solution that ingests and stores traces that provide maximum observability across your application estate.
However, sometimes constraints mean that storing all of your traces is not desirable, for example runtime or egress traffic related costs.
There are a number of ways to lower trace volume, including varying sampling strategies.

Sampling is the process of determining which traces to store (in Tempo or Grafana Cloud Traces) and which to discard. Sampling comes in two different strategy types: head and tail sampling.

Sampling functionality exists in both [Grafana Alloy](https://grafana.com/docs/alloy/) and the OpenTelemetry Collector. Alloy can collect, process, and export telemetry signals, with configuration files written in [Alloy configuration syntax](https://grafana.com/docs/alloy/<ALLOY_VERSION>/get-started/configuration-syntax/).

Refer to [Enable tail sampling](https://grafana.com/docs/tempo/<TEMPO_VERSION>/configuration/grafana-alloy/tail-sampling/enable-tail-sampling/) for instructions.

## Head and tail sampling

When sampling, you can use a head or tail sampling strategy.

With a head sampling strategy, the decision to sample the trace is usually made as early as possible and doesn’t need to take into account the whole trace.
It’s a simple but effective sampling strategy.
Refer to the [Head sampling documentation](https://grafana.com/docs/grafana-cloud/monitor-applications/application-observability/collector/sampling/head/#head-sampling) for [Application Observability](https://grafana.com/docs/grafana-cloud/monitor-applications/application-observability/) for more information.

With a tail sampling strategy, the decision to sample a trace is made after considering all or most of the spans. For example, tail sampling is a good option to sample only traces that have errors or traces with long request duration.
Tail sampling is more complex to configure, implement, and maintain but is the recommended sampling strategy for large systems with a high telemetry volume.

For more information about sampling, refer to the [OpenTelemetry Sampling](https://opentelemetry.io/docs/concepts/sampling/) documentation.

![Tail sampling overview and components with Tempo, Alloy, and Grafana](/media/docs/tempo/sampling/tempo-tail-based-sampling.svg)

## Sampling and telemetry correlation

Sampling is a decision on whether or not to keep (and then store) a trace, or whether to discard it.
These decisions have implications when it comes to correlating trace data with other signals.

For example, many services that are instrumented also produce logs, metrics, or profiles.
These signals can reference each other.
In the case of a trace, this reference can be via a trace ID embedded into a [log line](https://grafana.com/docs/grafana/<GRAFANA_VERSION>/datasources/tempo/traces-in-grafana/link-trace-id/), an [exemplar](https://grafana.com/docs/grafana/<GRAFANA_VERSION>/fundamentals/exemplars/) embedded into a metric value, or a profile ID [embedded into a trace](https://grafana.com/docs/grafana-cloud/monitor-applications/profiles/traces-to-profiles/).

By choosing to not sample a trace, a particular signal references the dropped trace's ID in some cases.
This drop can lead to a situation in Grafana where following a link to a trace ID from a log line or an exemplar from a metric value, results in a query for that trace ID failing because the trace has not been sampled.
Profiles may not show up without specifically querying for them, because a trace that would have included the profile's flame graph hasn't been stored.

This isn't usually a huge issue, because sampling policies tend to be chosen that show non-normative behavior, for example, errors being thrown or long latencies on requests.
An observer is more likely to be choosing traces that show these issues rather than the required behavior.
Understand how signals correlate between each other helps determine how to choose these policies.

## How tail sampling works in the OpenTelemetry Tail Sampling Processor

In tail sampling, sampling decisions are made at the end of the workflow allowing for a more accurate sampling decision.
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

When the first span for the trace is observed, the decision period time of 10 seconds is initiated.
After the decision period has expired, the tail sampler won't have observed any spans with an error status, and will therefore discard the trace spans.

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

| Cache type    | Use case                                                                                               | Benefits/Considerations                                                                                                                                                                       |
| ------------- | ------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Sample caches | Keep any future spans from traces that have been sampled.                                              | <ul><li>Cuts down span storage per trace to only those matching policies.</li><li> Can cause fragmented tracing.</li></ul>                                                                    |
| Non-sampled   | Drop any future spans from traces where a decision to not sample those traces has explicitly occurred. | <ul><li>Lowers chance of storing traces after the initial decision period. </li><li>Misses any trace whose spans exhibit future policy criteria matching.</li></ul>                           |
| Both          | Use an initial decision period that makes a decision once and uses that decision going forward.        | <ul><li>Guarantees capture of full traces.</li><li>Lower chance of capturing useful traces with a long duration.</li><li>Can lose spans if they are longer than the decision period.</li><ul> |

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
You can configure the exporter with targets using static IP addresses, multi-IP DNS A record entries, and a Kubernetes headless service resolver.
Using this configuration lets you scale up or down the number of layer two collectors.

There are some important points to note with the load balancer exporter around scaling and resilience, mostly around its eventual consistency model. For more information, refer to [Resilience and scaling considerations](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/exporter/loadbalancingexporter/README.md#resilience-and-scaling-considerations).
The most important in terms of tail sampling is that routing occurs based on an algorithm taking into account the number of backends available to the load balancer.
This can affect the target for trace ID spans before eventual consistency occurs.

For an example manifest for a two layer OpenTelemetry Collector deployment based around Kubernetes services, refer to the [Kubernetes service resolver README](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/exporter/loadbalancingexporter/example/k8s-resolver/README.md).