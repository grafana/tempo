---
title: Troubleshoot Tempo
menuTitle: Troubleshoot
description: Learn how to troubleshoot operational issues for Grafana Tempo.
weight: 700
aliases:
  - ../operations/troubleshooting/
---

# Troubleshoot Tempo

This section helps with day zero operational issues that may come up when getting started with Tempo.
The documents walk you through debugging each part of the ingestion and query pipeline to diagnose issues.

In addition, the [Tempo runbook](https://github.com/grafana/tempo/blob/main/operations/tempo-mixin/runbook.md) can help with remediating operational issues.

## Sending traces

- [Spans are being refused with "pusher failed to consume trace data"](max-trace-limit-reached/)
- [Is Grafana Alloy sending to the backend?](agent/)

## Querying

- [Unable to find my traces in Tempo](unable-to-see-trace/)
- [Error message "Too many jobs in the queue"](too-many-jobs-in-queue/)
- [Queries fail with 500 and "error using pageFinder"](bad-blocks/)
- [I can search traces, but there are no service name or span name values available](search-tag/)
- [Error message `response larger than the max (<number> vs <limit>)`](response-too-large/)

## Metrics-generator

- [Metrics or service graphs seem incomplete](metrics-generator/)