---
title: Monitoring
---

# Monitoring Tempo

The Tempo repository has a [mixin](https://github.com/grafana/tempo/tree/main/operations/tempo-mixin) that includes a
set of dashboards, rules and alerts. Together, these can be used to monitor Tempo in production.

## Dashboards

The Tempo mixin has four Grafana dashboards in the `yamls` folder that can be downloaded and imported into your Grafana UI.
Note that at the moment, these work well when Tempo is run in a k8s environment and metrics scraped have the
`cluster` and `namespace` labels!

### Tempo / Reads

> This is available as `tempo-reads.json`.

The Reads dashboard gives information information on Requests, Errors and Duration (R.E.D) on the Query Path of Tempo.
Each query touches the Gateway, Tempo-Query, Query-Frontend, Queriers, Ingesters, Cache (if present) and the backend.

Use this dashboard to monitor the performance of each of the above mentioned components and to decide the number of
replicas in each deployment.

### Tempo / Writes

> This is available as `tempo-writes.json`.

The Reads dashboard gives information information on Requests, Errors and Duration (R.E.D) on the write/ingest Path of Tempo.
A write query touches the Gateway, Distributors, Ingesters and eventually the backend. This dashboard also gives information
on the number of operations performed by the Compactor to the backend.

Use this dashboard to monitor the performance of each of the above mentioned components and to decide the number of
replicas in each deployment.

### Tempo / Resources

> This is available as `tempo-resources.json`.

The Resources dashboard provides information on `CPU`, `Container Memory` and `Go Heap Inuse`, and is useful for resource
provisioning for the different Tempo components.

Use this dashboard to see if any components are running close to their assigned limits!

### Tempo / Operational

> This is available as `tempo-operational.json`.

The Tempo Operational dashboard deserves special mention b/c it probably a stack of dashboard anti-patterns.
It's big and complex, doesn't use jsonnet and displays far too many metrics in one place.  And I love it.
For just getting started the Reads, Write and Resources dashboards are great places to learn how to monitor Tempo in an opaque way.

This dashboard is included in this repo for two reasons:

- It provides a stack of metrics for other operators to consider monitoring while running Tempo.
- We want it in our internal infrastructure and we vendor the tempo-mixin to do this.


## Rules and Alerts

The Rules and Alerts are available as [yaml files in the mixin](https://github.com/grafana/tempo/tree/main/operations/tempo-mixin/yamls) on the repository.

To set up alerting, download the provided json files and configure them for use on your Prometheus monitoring server.

Check the [runbook](https://github.com/grafana/tempo/blob/main/operations/tempo-mixin/runbook.md) to understand the
various steps that can be taken to fix firing alerts!
