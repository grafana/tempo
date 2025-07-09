---
description: "Learn how tracing data can help you understand application insights and performance as well as triaging issues in your services and applications."
keywords:
  - Grafana
  - traces
  - tracing
title: Use traces to find solutions
menuTitle: Solutions with traces
aliases:
  - ./introduction/solutions-with-traces/ # /docs/tempo/<TEMPO_VERSION>/introduction/solutions-with-traces/
weight: 150
---

# Use traces to find solutions

{{< shared id="traces-intro-2" >}}

Tracing is best used for analyzing the performance of your system, identifying bottlenecks, monitoring latency, and providing a complete picture of requests processing.

* Decrease mean time to repair and mean time to identify an issue by pinpointing exactly where errors or latency are occurring within a transaction across multiple services.
* Optimize bottlenecks and long-running code by visualizing the path and duration of requests. Tracing can help identify bottleneck operations and long-running pieces of code that could benefit from optimization.
* Detect issues with generated metrics. Tracing generates metrics related to request rate, error rate, and duration of requests. You can set alerts against these high-level signals to detect problems.
* Seamless telemetry correlation. Use tracing in conjunction with logs and metrics for a comprehensive view of events over time, during an active incident, or for root-cause analysis. Tracing shows relationships between services and dependencies.
* Monitor compliance with policies. Business policy adherence ensures that services are correctly isolated using generated metrics and generated service graphs.

{{< /shared >}}

Each use case provides real-world examples, including the background of the use case and how tracing highlighted and helped resolve any issues.