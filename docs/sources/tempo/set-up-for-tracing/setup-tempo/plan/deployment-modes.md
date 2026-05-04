---
title: Deployment modes
description: Choose between monolithic and microservices deployment modes for Tempo.
menuTitle: Deployment modes
weight: 200
---

# Deployment modes

Tempo supports two deployment modes: monolithic and microservices. All components are compiled into a single binary, and the `-target` flag determines which mode runs.

## Monolithic mode

In monolithic mode, the required components run in a single process using `-target=all`, which is the default. No Kafka is required. The distributor pushes trace data in-process directly to the live-store and metrics-generator, and traces are flushed to the configured storage backend. Object storage is recommended for production deployments.

Use monolithic mode when:

- You are getting started with Tempo or evaluating it
- You need a development or testing environment
- Your trace volume is under 25-35 MB/s or 55k-80k spans/s
- Operational simplicity matters more than independent scaling

Monolithic mode has some trade-offs to be aware of.
All components share the same resource pool, so a spike in query load can affect write throughput and vice versa.
There is no independent scaling: you can scale vertically or run multiple identical instances, but you cannot scale individual components separately.
At higher volumes, memory pressure from collocated components can cause issues.

## Microservices mode

In microservices mode, each component runs as a separate process with its own `-target` flag. For example, `-target=distributor` or `-target=querier`. This mode requires a Kafka-compatible system, such as Apache Kafka, Redpanda, or WarpStream, as the durable queue between the distributor and downstream consumers.

Use microservices mode when:

- You are running a production deployment
- You have high trace volumes that require independent scaling
- You need high availability and isolated failure domains
- You want to scale write throughput, query performance, and recent-data capacity independently

Microservices mode provides independent scaling for each component and isolated failure domains. A querier crash doesn't affect ingestion, and a block-builder restart doesn't affect query availability. Live-stores can be deployed across availability zones for high availability.

## Choosing a mode

| Consideration | Monolithic | Microservices |
|---|---|---|
| Kafka required | No | Yes |
| Scaling | Single process; scale vertically or run multiple identical instances | Each component scales independently |
| Failure isolation | All components share resources | Isolated failure domains per component |
| Operational complexity | Low | Higher, with more processes to manage |
| Best for | Getting started, development, up to 25-35 MB/s | Production, high volume, high availability |

## Next steps

- For detailed architecture, component descriptions, scaling guidelines, and migration guidance, refer to the [Deployment modes reference](/docs/tempo/<TEMPO_VERSION>/reference-tempo-architecture/deployment-modes/).
- To size your cluster, refer to [Size the cluster](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/plan/size/).
- To deploy Tempo, refer to [Deploy your Tempo instance](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/deploy/).
