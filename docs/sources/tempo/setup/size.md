---
title: Size the cluster
menuTitle: Size the cluster
description: Plan the size of your Tempo cluster.
aliases:
  - /docs/tempo/deployment
  - /docs/tempo/deployment/deployment
  - /docs/tempo/setup/deployment
weight: 250
---

# Size the cluster

Resource requirements for your Grafana Tempo cluster depend on the amount and rate of data processed, retained, and queried.

This document provides basic configuration guidelines that you can use as a starting point to help size your own deployment.

{{< admonition type="note" >}}
Tempo is under continuous development. These requirements can change with each release.
{{< /admonition >}}

## Factors impacting cluster sizing

The size of the cluster you deploy depends on how many resources it needs for a given ingestion rate and retention: number of spans/time, average byte span size, rate of querying, and retention N days.

Tracing instrumentation also effects your Tempo cluster requirements.
Refer to [Best practices](https://grafana.com/docs/tempo/<TEMPO_VERSION>/getting-started/best-practices/) for suggestions on determining where to add spans, span length, and attributes.

## Example sample cluster sizing

Distributor:

* 1 replica per every 10MB/s of received traffic
* CPU: 2 cores
* Mem: 2 GB

Ingester:

* 1 replica per every 3-5MB/s of received traffic.
* CPU: 2.5 cores
* Mem: 4-20GB, determined by trace composition

Querier:

* 1 replica per every 1-2MB/s of received traffic.
* CPU: dependent on trace size and queries
* Mem: 4-20GB, determined by trace composition and queries
* This number of queriers should give good performance for typical search patterns and time ranges. Can scale up or down to fit the specific workload.

Query-Frontend:

* 2 replicas, for high availability
* CPU: dependent on trace size and queries
* Mem: 4-20GB, dependent on trace size and queries

Compactor:

* 1 replica per every 3-5 MB/s of received traffic.
* CPU: 1 core (compactors are primarily I/O bound, therefore do not require much CPU)
* Mem: 4-20GB, determined by trace composition

## Performance tuning resources

Refer to these documents for additional information on tuning your Tempo cluster:

* [Monitor Tempo](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/monitor/)
* [Tune search performance](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/backend_search/)
* [Improve performance with caching](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/caching/)
* [Dedicated attribute columns](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/dedicated_columns/)

For information on more advanced system options, refer to [Manage advanced systems](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/manage-advanced-systems/).