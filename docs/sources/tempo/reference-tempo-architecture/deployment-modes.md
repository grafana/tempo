---
title: Deployment modes
menuTitle: Deployment modes
description: Monolithic and microservices deployment modes in detail.
weight: 400
topicType: concept
versionDate: 2026-03-20
---

# Deployment modes

Tempo can be deployed in monolithic or microservices mode. Microservices mode requires a Kafka-compatible system. Monolithic mode doesn't use Kafka.

_Monolithic mode_ was previously called _single binary mode_.

{{< admonition type="note" >}}
The previous _scalable monolithic mode_, also known as _scalable single binary mode_ or SSB, has been removed in v3.0.
{{< /admonition >}}

All components are compiled into the same binary. The `-target` command-line parameter, or `target` in configuration, determines which components run in a given process. The default target is `all`, which is the monolithic deployment mode.

```bash
tempo -target=all
```

Refer to the [command line flags](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/command-line-flags/) documentation for more information on the `-target` flag.

## Monolithic mode

In monolithic mode, the required components run in a single process using `-target=all`, which is the default. Components that are only needed in microservices mode, such as the block-builder, are excluded.

No Kafka is required. The distributor pushes trace data in-process directly to the live-store and metrics-generator. Traces are flushed to the configured storage backend without an intermediate message queue. Object storage is recommended for production deployments.

### When to use monolithic mode

Monolithic mode is suitable for getting started, development environments, and low to moderate trace volumes where operational simplicity matters more than independent scaling.

### Limitations

Components share the same resource pool. A spike in query load can affect write throughput and vice versa. There is no independent scaling. You can run multiple monolithic instances, but each instance runs the same set of components. At higher volumes, memory pressure from collocated components, particularly the live-store and querier, can cause out-of-memory issues.

### Resource considerations

Monolithic instances need enough memory to handle the live-store's in-memory trace buffer, the querier's concurrent job execution, and the backend worker's memory for block merging. As volume increases, the instance is limited by whichever component is most resource-hungry.

### Example

Refer to [Docker Compose examples in the Tempo repository](https://github.com/grafana/tempo/tree/main/example/docker-compose/) for sample deployments.

For an annotated example configuration for Tempo, refer to the [Introduction to MLTP](https://github.com/grafana/intro-to-mltp) repository, which includes a [sample `tempo.yaml`](https://github.com/grafana/intro-to-mltp/blob/main/tempo/tempo.yaml) for a monolithic instance.

## Microservices mode

In microservices mode, each component runs as a separate process with its own `-target`. This is the recommended mode for production.

The configuration associated with each component's deployment specifies a `target`. For example, to deploy a `querier`, the configuration would contain `target: querier`. A command-line deployment may specify the `-target=querier` flag.

### When to use microservices mode

Use microservices mode for production deployments, high trace volumes requiring independent scaling, and environments where high availability is important.

### Advantages

Microservices mode provides independent scaling. You can scale block-builders for write throughput, queriers for query performance, and live-stores for recent data capacity, all independently. Failure domains are isolated: a querier OOM doesn't affect data ingestion, and a block-builder restart doesn't affect query availability. Live-stores can be deployed across availability zones for high availability. Each component gets exactly the resources it needs, avoiding the over-provisioning required in monolithic mode.

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

Adding or removing instances of any component doesn't require reconfiguring other components, aside from Kafka partition management.

### Example

Refer to the [distributed Docker Compose example](https://github.com/grafana/tempo/tree/main/example/docker-compose/distributed) in the Tempo repository.

## Components by deployment mode

Not all components and configuration blocks apply to both modes. The following table summarizes which components run in each mode and how shared components differ.

| Component | Config block | Monolithic | Microservices |
|---|---|---|---|
| Distributor | `distributor` | Pushes data in-process to the live-store and metrics-generator | Writes data to Kafka |
| Ingest | `ingest` | Not used | Kafka connection settings for the write path |
| Block-builder | `block_builder` | Not used | Consumes from Kafka, builds Parquet blocks, flushes to object storage |
| Live-store | `live_store` | Receives data directly from the distributor | Consumes from Kafka |
| Live-store client | `live_store_client` | Querier-to-live-store client (runs in-process) | gRPC client for querier-to-live-store communication |
| Query-frontend | `query_frontend` | Runs in-process | Runs as a separate process |
| Querier | `querier` | Runs in-process | Runs as a separate process |
| Backend scheduler | `backend_scheduler` | Runs in-process | Runs as a separate process |
| Backend worker | `backend_worker` | Runs in-process | Runs as a separate process |
| Metrics-generator | `metrics_generator` | Optional, runs in-process | Optional, runs as a separate process |
| Storage | `storage` | Storage backend for trace data (object storage recommended; local supported for dev/test) | Object storage for trace data |
| Memberlist | `memberlist` | Cluster membership | Cluster membership |
| Overrides | `overrides` | Per-tenant limits | Per-tenant limits |

For full configuration details, refer to the [configuration reference](/docs/tempo/<TEMPO_VERSION>/configuration/).

## Migrating between modes

Moving from monolithic to microservices mode involves deploying individual components with appropriate `-target` flags, pointing all components at the same Kafka cluster, object storage, and memberlist, then scaling down the monolithic instances.

Since Kafka provides durability, no data is lost during the transition. Live-stores replay from Kafka on startup, and block-builders continue from their last committed offset.

## Related resources

Refer to [Plan your Tempo deployment](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/plan/) for deployment planning and sizing guidance.
