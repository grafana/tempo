---
title: Deployment modes
menuTitle: Deployment modes
description: Monolithic and microservices deployment modes in detail.
weight: 400
topicType: concept
versionDate: 2026-03-20
---

# Deployment modes

Tempo supports two deployment modes: monolithic and microservices. Both require a Kafka-compatible system.

All components are compiled into the same binary. The `-target` parameter (or `target` in configuration) determines which components run in a given process.

## Monolithic mode

In monolithic mode, all components run in a single process using `-target=all` (the default).

This means one process handles distributor, block-builder, live-store, querier, query-frontend, backend scheduler, and backend worker responsibilities. It reads from and writes to Kafka just like a microservices deployment.

### When to use monolithic mode

Monolithic mode is suitable for getting started, development environments, and low to moderate trace volumes where operational simplicity matters more than independent scaling.

### Limitations

All components share the same resource pool. A spike in query load can affect write throughput and vice versa. There's no independent scaling — you can run multiple monolithic instances, but each runs every component. At higher volumes, memory pressure from collocated components (particularly live-store and querier) can cause OOM issues.

### Resource considerations

Monolithic instances need enough memory to handle the live-store's in-memory trace buffer, the querier's concurrent job execution, the block-builder's scratch space, and the compactor's memory for block merging. As volume increases, the instance becomes bottlenecked by whichever component is most resource-hungry.

## Microservices mode

In microservices mode, each component runs as a separate process with its own `-target`. This is the recommended mode for production.

### When to use microservices mode

Use microservices mode for production deployments, high trace volumes requiring independent scaling, and environments where high availability is important.

### Advantages

Microservices mode provides independent scaling — you can scale block-builders for write throughput, queriers for query performance, and live-stores for recent data capacity, all independently. Failure domains are isolated: a querier OOM doesn't affect data ingestion, and a block-builder restart doesn't affect query availability. Live-stores can be deployed across availability zones for high availability. Each component gets exactly the resources it needs, avoiding the over-provisioning required in monolithic mode.

### Component scaling guidelines

| Component | Scaling strategy | Notes |
|---|---|---|
| Distributor | Horizontal | Stateless. Scale based on ingestion rate. |
| Block-builder | Horizontal | Bounded by Kafka partition count. Scale based on data volume. |
| Live-store | Horizontal | Bounded by Kafka partition count. Scale based on recent data query volume and memory. |
| Query frontend | Vertical | Keep to 2 replicas. Scale up CPU/RAM rather than adding replicas. |
| Querier | Horizontal | Scale based on query concurrency and latency requirements. |
| Backend worker | Horizontal | Handles compaction and retention. Scale based on block count and compaction lag. |
| Metrics-generator | Horizontal | Scale based on trace volume and number of generated series. |

### Kafka as the connecting fabric

In microservices mode, Kafka is the primary communication channel for the write path. Components don't communicate directly for data transfer. Distributors write to Kafka. Block-builders, live-stores, and metrics-generators each consume from Kafka independently. Queriers contact live-stores over gRPC for recent data. All components access object storage for block data.

Adding or removing instances of any component doesn't require reconfiguring other components (beyond Kafka partition management).

## Migrating between modes

Moving from monolithic to microservices mode involves deploying individual components with appropriate `-target` flags, pointing all components at the same Kafka cluster, object storage, and memberlist, then scaling down the monolithic instances.

Since Kafka provides durability, no data is lost during the transition. Live-stores replay from Kafka on startup, and block-builders continue from their last committed offset.

## Related resources

Refer to the [deployment modes setup guide](https://grafana.com/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/plan/deployment-modes/) for configuration details.
