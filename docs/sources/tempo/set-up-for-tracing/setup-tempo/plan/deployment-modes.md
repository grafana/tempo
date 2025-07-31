---
title: Monolithic and microservices modes
description: Learn about the different deployment modes for Tempo
menuTitle: Deployment modes
weight: 200
---

# Monolithic and microservices modes

Tempo can be deployed in _monolithic_ or _microservices_ modes.

_Monolithic mode_ was previously called _single binary mode_. Similarly _scalable monolithic mode_ was previously called _scalable single binary mode_.
While the documentation reflects this change, some URL names and deployment tooling may not yet reflect this change.

The deployment mode is determined by the runtime configuration `target`, or
by using the `-target` flag on the command line. The default target is `all`,
which is the monolithic deployment mode.

```bash
tempo -target=all
```

Refer to the [Command line flags](../../command-line-flags/) documentation for more information on the `-target` flag.

## Monolithic mode

Monolithic mode uses a single Tempo binary is executed, which runs all of the separate components within a single running process.
This means that a single instance both ingests, stores, compacts and queries trace data.

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

### Scaling monolithic mode

Monolithic mode can be horizontally scaled out.
This scalable monolithic mode is similar to the monolithic mode in that all components are run within one process.
Horizontal scale out is achieved by instantiating more than one process, with each having `-target` set to `scalable-single-binary`.

This mode offers some flexibility of scaling without the configuration complexity of the full
microservices deployment.

Each of the `queriers` perform a DNS lookup for the `frontend_address` and connect to the addresses found within the DNS record.

#### Example

Find a docker-compose deployment example at [https://github.com/grafana/tempo/tree/main/example/docker-compose/scalable-single-binary](https://github.com/grafana/tempo/tree/main/example/docker-compose/scalable-single-binary)

## Microservices mode

Microservices mode is a configuration that allows for a fully horizontally scaled deployment that allows individual components of Tempo to be given a set number of replicas.

In microservices mode, components are deployed in distinct processes.
Scaling is per component, which allows for greater flexibility in scaling and more
granular failure domains. This is the preferred method for a production
deployment, but it's also the most complex.

Each instance of a component is a single process, therefore dedicating themselves to part of the Tempo infrastructure (as opposed to the Monolithic mode where each process runs several different Tempo components).

This allows for:

- A more resilient deployment that includes data replication factors. Components can be run over multiple nodes, such as in a Kubernetes cluster, ensuring that catastrophic failure of one node does not have a failure impact for the system as a whole. For example, by default, three independent ingesters are all sent the same span data by a distributor. If two of those ingesters fail, the data is still processed and stored by an ingester.
- Horizontal scaling up and down of clusters. For example, an organization may see upticks in traffic in certain periods (say, Black Friday), and need to scale up the amount of trace data being ingested for a week. Microservices mode allows them to temporarily scale up the number of ingesters, queriers, etc. that they may need with no adverse impact on the overall system which may not be as simple with Monolithic or SSB mode.
- However, much like the difference between Monolithic mode and SSB mode, there is an increased TCO and maintenance cost that goes along with Microservices mode. Whilst it is more flexible, it requires more attention to run proficiently. Microservices mode is the default deployment for Tempo and Grafana Enterprise Traces (GET) via the tempo-distributed Helm Chart.

The configuration associated with each component's deployment specifies a
`target`. For example, to deploy a `querier`, the configuration would contain
`target: querier`. A command-line deployment may specify the `-target=querier`
flag.

Each of the components referenced in [Architecture](/docs/tempo/<TEMPO_VERSION>/introduction/architecture/) must be deployed for a working Tempo instance.

![Microservices mode architecture](/media/docs/tempo/architecture/tempo-TempoMicroservices-arch.png)

### Example

Find a docker-compose deployment example at [https://github.com/grafana/tempo/tree/main/example/docker-compose/distributed](https://github.com/grafana/tempo/tree/main/example/docker-compose/distributed).
