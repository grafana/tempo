---
description: "Learn how tracing data can help you understand application insights and performance as well as triaging issues in your services and applications."
keywords:
  - Grafana
  - traces
  - tracing
title: Use traces to find solutions
menuTitle: Solutions with traces
weight: 300
---

# Use traces to find solutions

Tracing is best used for analyzing the performance of your system, identifying bottlenecks, monitoring latency, and providing a complete picture of how requests are processed.

* Decrease MTTR/MTTI: Tracing helps reduce Mean Time To Repair (MTTR) and Mean Time To Identify (MTTI) by pinpointing exactly where errors or latency are occurring within a transaction across multiple services.
*  Optimization of bottlenecks and long-running code: By visualizing the path and duration of requests, tracing can help identify bottleneck operations and long-running pieces of code that could benefit from optimization.
*  Metrics generation and RED signals: Tracing can help generate useful metrics related to Request rate, Error rate, and Duration of requests (RED). You can set alerts against these high-level signals to detect problems when they arise.
* Seamless telemetry correlation: Using tracing in conjunction with logs and metrics can help give you a comprehensive view of events over time during an active incident or postmorterm analysis by showing relationships between services and dependencies.
* Monitor compliance with policies: Business policy adherence ensures that services are correctly isolated using generated metrics and generated service graphs.


Each use case provides real-world examples, including the background of the use case and how tracing highlighted and helped resolve any issues.