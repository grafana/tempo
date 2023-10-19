---
aliases:
- /docs/tempo/v1.2.1/grafana-agent/tail-based-sampling/
title: Tail-based sampling
---

# Tail-based sampling

Tempo aims to provide an inexpensive solution that makes 100% sampling possible.
However, sometimes constraints will make a lower sampling percentage necessary or desirable,
such as runtime or egress traffic related costs.
Probabilistic sampling strategies are easy to implement,
but also run the risk of discarding relevant data that you'll later want.

In tail-based sampling, sampling decisions are made at the end of the workflow allowing for a more accurate sampling decision.
The Grafana Agent groups span by trace ID and check its data to see
if it meets one of the defined policies (for example, latency or status_code).
For instance, a policy can check if a trace contains an error or if it took
longer than a certain duration.

A trace will be sampled if it meets at least one policy.

To group spans by trace ID, the Agent buffers spans for a configurable amount of time,
after which it will consider the trace complete.
Longer running traces will be split into more than one.
However, waiting longer times will increase the memory overhead of buffering.

One particular challenge of grouping trace data is for multi-instance Agent deployments,
where spans that belong to the same trace can arrive to different Agents.
To solve that, you can configure the Agent to load balance traces across agent instances
by exporting spans belonging to the same trace to the same instance.

This is achieved by redistributing spans by trace ID once they arrive from the application.
The Agent must be able to discover and connect to other Agent instances where spans for the same trace can arrive.
For kubernetes users, that can be done with a [headless service](https://kubernetes.io/docs/concepts/services-networking/service/#headless-services).

Redistributing spans by trace ID means that spans are sent and received twice,
which can cause a significant increase in CPU usage.
This overhead will increase with the number of Agent instances that share the same traces.

<p align="center"><img src="../tail-based-sampling.png" alt="Tail-based sampling overview"></p>

## Quickstart

To start using tail-based sampling, define a sampling policy.
If you're using a multi-instance deployment of the agent,
add load balancing and specify the resolving mechanism to find other Agents in the setup.
To see all the available config options, refer to the [configuration reference](https://github.com/grafana/agent/blob/main/docs/configuration/tempo-config.md).

```
tempo:
  configs:
    - name: default
      ...
      tail_sampling:
        policies:
          # sample traces that have a total duration longer than 100ms
          - latency:
              threshold_ms: 100
          # sample traces that contain at least one span with status code ERROR
          - status_code:
              status_codes:
                - "ERROR"
      load_balancing:
        resolver:
          dns:
            hostname: host.namespace.svc.cluster.local
```
