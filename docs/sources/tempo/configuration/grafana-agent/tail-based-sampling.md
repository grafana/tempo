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
To see all the available configuration options, refer to the [configuration reference](/docs/agent/latest/configuration/traces-config/).

```yaml
traces:
  configs:
    - name: default
      ...
      tail_sampling:
        policies:
          # sample traces that have a total duration longer than 100ms
          - type: latency
            latency:
              threshold_ms: 100
          # sample traces that contain at least one span with status code ERROR
          - type: status_code
            status_code:
              status_codes:
                - "ERROR"
      load_balancing:
        resolver:
          dns:
            hostname: host.namespace.svc.cluster.local
```

## Examples

Sampling logic can be summarized into two rules:

1. If a policy is met, the trace is sampled. Even if other policies are not met.
2. If any of the spans of the trace meet the policy requirements, the trace is sampled.

Next, there are a couple of examples of `tail_sampling` configurations,
with descriptions of the policies and the expected behavior.

### Sampling by latency and status

Sampling traces that have a total duration longer than 100ms and traces that contain at least one span with status code `ERROR`.
Total duration is determined by looking at the earliest start time and latest end time.

```yaml
traces:
  configs:
    - name: default
      ...
      tail_sampling:
        policies:
          - type: latency
            latency:
              threshold_ms: 100
          - type: status_code
            status_code:
              status_codes:
                - "ERROR"
```

### Sampling by attributes

Sampling traces that do not contain the attribute `http.endpoint` equal to `/status` and `/metrics`.

```yaml
traces:
  configs:
    - name: default
      ...
      tail_sampling:
        policies:
        - attributes:
          - name: http.endpoint
            values:
              - /status
              - /metrics
            invert_match: true
```

### Sampling with `and` policy

Sampling traces that have an attribute `http.endpoint` that matches `/api/v1/*`
and that have a total duration longer than 100ms.

```yaml
traces:
  configs:
    - name: default
      ...
      tail_sampling:
      policies:
      - type: and
        and:
          and_sub_policy:
            - type: string_attribute
              string_attribute:
              - name: http.endpoint
                value: /api/v1/*
                enabled_regex_matching: true
                cache_max_size: 10
            - type: latency
              latency:
                threshold_ms: 100
```

### Multi-requirement sampling

Sampling requirements are the following:

- Service A: latency> 3s
- Service B: only error spans or latency> 5s
- Service C: all spans

```yaml
traces:
  configs:
    - name: default
      ...
      tail_sampling:
        policies:
        # Service A: latency> 3s
        - type: and
          and:
            and_sub_policy:
              - type: latency
                name: latency
                latency:
                  threshold_ms: 3000
              - type: string_attribute
                name: service-name
                string_attribute:
                  name: service.name
                  values:
                    - serviceA
        # Service B requires two and policies
        # 1. spans with status code ERROR
        - type: and
          and:
            and_sub_policy:
              - type: status_code
                name: status_code
                status_code:
                  status_codes:
                    - "ERROR"
              - type: string_attribute
                name: service-name
                string_attribute:
                  name: service.name
                  values:
                    - serviceB
        # 2. latency> 5s
        - type: and
          and:
            and_sub_policy:
              - type: latency
                name: latency
                latency:
                  threshold_ms: 5000
              - type: string_attribute
                name: service name
                string_attribute:
                  name: service.name
                  values:
                    - serviceB
        # Service C: all spans
        - type: string_attribute
          string_attribute:
            name: service.name
            values:
              - serviceC
```

## Sampling based on k8s metadata

In this example, the Agent will sample traces that come from pods from the namespace `default`.
Via `scrape_configs`, spans are relabeled with kubernetes metadata,
in this case injecting the namespace attribute.

```yaml
traces:
  configs:
    - name: default
    ...
    scrape_configs:
      - job_name: kubernetes-pods
        kubernetes_sd_configs:
          - role: pod
        relabel_configs:
          - action: replace
            source_labels:
              - __meta_kubernetes_namespace
            target_label: namespace
    tail_sampling:
      policies:
      - type: string_attribute
        string_attribute:
          name: namespace
          values:
          - default
```
