---
title: Troubleshoot Tempo
menuTitle: Troubleshoot Tempo
description: Learn how to troubleshoot operational issues for Grafana Tempo.
weight: 200
aliases:
- /docs/tempo/troubleshooting/
---

# Troubleshoot Tempo

This section helps with day zero operational issues that may come up when getting started with Tempo.
The documents walk you through debugging each part of the ingestion and query pipeline to diagnose issues.

In addition, the [Tempo runbook](https://github.com/grafana/tempo/blob/main/operations/tempo-mixin/runbook.md) can help with remediating operational issues.

## Sending traces

- [Spans are being refused with "pusher failed to consume trace data"]({{< relref "./max-trace-limit-reached" >}})
- [Is the Grafana Agent sending to the backend?]({{< relref "./agent" >}})

## Querying

- [Unable to find my traces in Tempo]({{< relref "./unable-to-see-trace" >}})
- [Error message "Too many jobs in the queue"]({{< relref "./too-many-jobs-in-queue" >}})
- [Queries fail with 500 and "error using pageFinder"]({{< relref "./bad-blocks" >}})
- [I can search traces, but there are no service name or span name values available]({{< relref "./search-tag" >}})
- [Error message `response larger than the max (<number> vs <limit>)`]({{< relref "./response-too-large" >}})

## Metrics Generator

- [Metrics or service graphs seem incomplete]({{< relref "./metrics-generator" >}})