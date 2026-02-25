---
title: Monolithic and microservices modes
description: Learn about the different deployment modes for Tempo
menuTitle: Deployment modes
weight: 200
---

# Monolithic and microservices modes

Tempo can be deployed in _monolithic_ or _microservices_ modes.

_Monolithic mode_ was previously called _single binary mode_.

{{< admonition type="note" >}}
Tempo v3.0 requires a Kafka-compatible system for both monolithic and microservices modes. The previous _scalable monolithic mode_ (formerly called _scalable single binary mode_ or SSB) has been removed in v3.0.
{{< /admonition >}}

The deployment mode is determined by the runtime configuration `target`, or
by using the `-target` flag on the command line. The default target is `all`,
which is the monolithic deployment mode.

```bash
tempo -target=all
```

Refer to the [Command line flags](../../command-line-flags/) documentation for more information on the `-target` flag.

## Monolithic mode

Monolithic mode uses a single Tempo binary that runs all of the separate components within a single process.
This means that a single instance handles the distributor, block-builder, live-store, querier, query-frontend, backend-scheduler, and backend-worker roles.
The instance writes to and reads from a Kafka-compatible system for trace ingestion and retrieval.

Monolithic mode handles modest volumes of trace data without issues given a modest amount of resource.

However, when increased, sustained trace volume is sent to it, the monolithic deployment can incur problems in terms of resource usage, as more data is required to be indexed, stored and compacted.
This can lead in best cases to a 'running hot' and laggy experience and in worst cases cause OOM (Out Of Memory) and stalling issues, leading to crashing and restarting.
In this case, a single process deployment would lead to missed trace data.

To enable this mode, `-target=all` is used, which is the default.

Refer to [Architecture](/docs/tempo/<TEMPO_VERSION>/introduction/architecture/) for descriptions of the components.

![Monolithic mode architecture](/media/docs/tempo/architecture/tempo-TempoSingleBinary-arch.png)

### Example

Find docker-compose deployment examples in the tempo repository: [https://github.com/grafana/tempo/tree/main/example/docker-compose](https://github.com/grafana/tempo/tree/main/example/docker-compose/)

To see an annotated example configuration for Tempo, the [Introduction To MLTP](https://github.com/grafana/intro-to-mltp) example repository contains a [configuration](https://github.com/grafana/intro-to-mltp/blob/main/tempo/tempo.yaml) for a monolithic instance.

## Microservices mode

Microservices mode is a configuration that allows for a fully horizontally scaled deployment that allows individual components of Tempo to be given a set number of replicas.

In microservices mode, components are deployed in distinct processes.
Scaling is per component, which allows for greater flexibility in scaling and more
granular failure domains. This is the preferred method for a production
deployment, but it's also the most complex.

Each instance of a component is a single process, therefore dedicating themselves to part of the Tempo infrastructure (as opposed to the Monolithic mode where each process runs several different Tempo components).

This allows for:

- A more resilient deployment with high availability. Components can be run over multiple nodes, such as in a Kubernetes cluster, ensuring that catastrophic failure of one node does not have a failure impact for the system as a whole. Durability is provided by Kafka, which serves as the write-ahead log. Live-stores can be deployed across multiple availability zones for query availability.
- Horizontal scaling up and down of clusters. For example, an organization may see upticks in traffic in certain periods (say, Black Friday), and need to scale up the amount of trace data being ingested for a week. Microservices mode allows them to temporarily scale up the number of block-builders, live-stores, queriers, etc. that they may need with no adverse impact on the overall system.
- However, microservices mode has an increased TCO and maintenance cost compared to monolithic mode. While it is more flexible and scalable, it requires more attention to run proficiently. Microservices mode is the default deployment for Tempo and Grafana Enterprise Traces (GET) via the tempo-distributed Helm Chart.

The configuration associated with each component's deployment specifies a
`target`. For example, to deploy a `querier`, the configuration would contain
`target: querier`. A command-line deployment may specify the `-target=querier`
flag.

Each of the components referenced in [Architecture](/docs/tempo/<TEMPO_VERSION>/introduction/architecture/) must be deployed for a working Tempo instance.

![Microservices mode architecture](/media/docs/tempo/architecture/tempo-TempoMicroservices-arch.png)

### Example

Find a docker-compose deployment example at [https://github.com/grafana/tempo/tree/main/example/docker-compose/distributed](https://github.com/grafana/tempo/tree/main/example/docker-compose/distributed).
