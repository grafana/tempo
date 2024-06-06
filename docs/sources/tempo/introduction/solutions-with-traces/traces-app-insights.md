---
description: "Learn how tracing data can help you understand application insights and performance as well as triaging issues in your services and applications."
keywords:
  - Grafana
  - traces
  - tracing
title: Identify bottlenecks and establish SLOs
menuTitle: Identify bottlenecks and establish SLOs
weight: 320
---

# Identify bottlenecks and establish SLOs

Traces give a window into an applications performance, and the insights that go along with that. Using traces, you can discover bottlenecks and potential optimizations that can lead to decreased latency and response times, improving the number of requests the application can handle per second.

## Meet Handy Site Corp

Handy Site Corp, a fake website company, runs an ecommerce application that includes user authentication, a product catalog, order management, payment processing, and other services.

### Define realistic SLOs

Handy Site’s engineers need to establish service level objectives (SLOs) around latency to measure the latency customers experience with the checkout service.
To do this, they can leverage the metrics generated from their span data.

They need to establish realistic targets based on previous history of normal operation modes.
This data helps them identify degradation of service over time.
In addition, they want to be alerted when significant deviations occur.

### Utilize span metrics

After evaluating options, they decide to use [span metrics](https://grafana.com/docs/tempo/latest/metrics-generator/span_metrics/) as a service level indicator (SLI) to measure SLO compliance.

![Metrics generator and exemplars](/media/docs/tempo/intro/traces-metrics-gen-exemplars.png)

Tempo can generate metrics using the [metrics-generator component](https://grafana.com/docs/tempo/latest/metrics-generator/).
These metrics are created based on spans from incoming traces and demonstrate immediate usefulness with respect to application flow and overview.
This includes rate, error, and duration (RED) metrics.

Span metrics can provide in-depth monitoring of your system. The generated metrics show application-level insight into your monitoring, as traces propagate through your application's services.

Span metrics lower the entry barrier for using exemplars.
An [exemplar](https://grafana.com/docs/grafana/latest/basics/exemplars/) serves as a detailed example of one of the observations aggregated into a metric. An exemplar contains the observed value together with an optional timestamp and arbitrary trace IDs, which are typically used to reference a trace.
Since traces and metrics co-exist in the metrics-generator, exemplars can be automatically added, providing additional value to these metrics.

### Monitor latency

In this case, Handy Site wants to monitor latency.
They can leverage the `traces_spanmetrics_latency` metric with the corresponding labels, such as `service name = checkoutservice`.
For example, if there are specific spans of interest such as release versions, per-service endpoints, and others, you can create and use those metrics as extra dimensions for correlations.

With all of this in place, Handy Site uses the data generated with metrics-generator in [Grafana SLO](https://grafana.com/docs/grafana-cloud/alerting-and-irm/slo/) to establish an [SLI](https://grafana.com/docs/grafana-cloud/alerting-and-irm/slo/create/).
They can now be alerted to degradations in service quality that directly impacts their end user experience.

![Latency SLO dashboard](/media/docs/tempo/intro/traces-metrics-gen-SLO.png)