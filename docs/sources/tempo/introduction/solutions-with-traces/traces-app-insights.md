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

Traces are especially powerful when:

* Identify cause of bottlenecks usings application insights and performance
* Diagnose 500 errors and reduce MTTR

Each use case provides real-world examples, including the background of the use case and how tracing highlighted and helped resolve any issues.

## Focus on application insights and performance

Handy Site Corp, a fake website company, runs an ecommerce application that includes user authentication, a product catalog, order management, payment processing, and other services.

### Problem

Handy Site’s engineers have been tasked with establishing some service level objectives (SLOs) around latency to measure the latency customers experience with the checkout service.
To do this, they can leverage the metrics generated from their span data.

They need to establish realistic targets based on previous history of normal operation modes.
This data helps them identify degradation of service over time.
In addition, they want to be alerted when significant deviations occur.

### Solution

After evaluating options, they decide to use [span metrics](https://grafana.com/docs/tempo/latest/metrics-generator/span_metrics/) as a service level indicator (SLI) to measure SLO compliance.
Tempo can generate metrics using the [metrics-generator component](https://grafana.com/docs/tempo/latest/metrics-generator/).
These metrics are created based on spans from incoming traces and demonstrate immediate usefulness with respect to application flow and overview.
This includes rate, error, and duration (RED) metrics.

Span metrics can provide in-depth monitoring of your system. The generated metrics will show application-level insight into your monitoring, as far as tracing gets propagated through your applications.

Span metrics lower the entry barrier for using exemplars.
An [exemplar](https://grafana.com/docs/grafana/latest/basics/exemplars/) is a specific trace representative of measurement taken in a given time interval.
Since traces and metrics co-exist in the metrics-generator, exemplars can be automatically added, providing additional value to these metrics.

In this case, Handy Site wants to monitor latency.
They can leverage the`traces_spanmetrics_latency` metric with the corresponding labels, such as `service name = checkoutservice`.
For example, if there are specific spans of interest such as release versions, per-service endpoints, and others, you can create and use those metrics as extra dimensions for correlations.

With all of this in place, Handy Site used the data generated with metrics-generator in [Grafana SLO](https://grafana.com/docs/grafana-cloud/alerting-and-irm/slo/) to establish an [SLI](https://grafana.com/docs/grafana-cloud/alerting-and-irm/slo/create/).
They can now be alerted to degradations in service quality that directly impacts their end user experience.

SCREENSHOT