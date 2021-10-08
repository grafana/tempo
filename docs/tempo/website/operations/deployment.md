---
title: Deployment
aliases:
  - /docs/tempo/latest/deployment
  - /docs/tempo/latest/deployment/deployment
weight: 3
---

# Deploying Tempo

Tempo can be easily deployed through a number of tools as explained in this document.

> **Note**: The Tanka and Helm examples are equivalent.
> They are both provided for people who prefer different configuration mechanisms.

## Tanka / Jsonnet

The Jsonnet files that you need to deploy Tempo with Tanka are available here:

- [single binary](https://github.com/grafana/tempo/tree/main/operations/jsonnet/single-binary)
- [microservices](https://github.com/grafana/tempo/tree/main/operations/jsonnet/microservices)

Here are a few [examples](https://github.com/grafana/tempo/tree/main/example/tk) that use official Jsonnet files.
They display the full range of configurations available to Tempo.

## Helm

Helm charts are available in the grafana/helm-charts repo:

- [single binary](https://github.com/grafana/helm-charts/tree/main/charts/tempo)
- [microservices](https://github.com/grafana/helm-charts/tree/main/charts/tempo-distributed)

## Kubernetes manifests

You can find a collection of Kubernetes manifests to deploy Tempo in the
[operations/kube-manifests](https://github.com/grafana/tempo/tree/main/operations/kube-manifests)
folder.  These are generated using the Tanka / Jsonnet.

# Deployment scenarios

Tempo can be deployed in one of three modes.

- Single binary
- Scalable single binary
- Microservices

Which mode is deployed is determined by the runtime configuration `target`, or
by using the `-target` flag on the command line. The default target is `all`,
which is the single binary deployment mode.

## Single binary

A single binary mode deployment runs all top-level components in a single
process, forming an instance of Tempo.  The single binary mode is the simplest
implement.  Refer to [Architecture]({{< relref "./architecture" >}}) for
descriptions of the components.

Find docker-compose deployment examples at:

- [https://github.com/grafana/tempo/tree/main/example/docker-compose/local](https://github.com/grafana/tempo/tree/main/example/docker-compose/local)
- [https://github.com/grafana/tempo/tree/main/example/docker-compose/s3](https://github.com/grafana/tempo/tree/main/example/docker-compose/s3)

## Scalable single binary

A scalable single binary mode deployment may have more than one single binary
mode Tempo instance.  Matching components, such as the distributors, within
each instance are aware of each other; this is known as clustering.  The
components form a [consistent hash ring]({{< relref "./consistent_hash_ring"
>}}) to coordinate operations among the multiple Tempo instances.

The configuration of a scalable single binary defines `kvstore`.  Here is an
example `memberlist` configuration:

```yaml
target: scalable-single-binary
ingester:
  lifecycler:
    ring:
      kvstore:
        store: memberlist
```

Additionally, the `queriers` must know the DNS name that will contain the addresses of all other instances.
For example:

```yaml
querier:
  frontend_worker:
    frontend_address: tempo.lab.example.com:9095
```

Each of the `queriers` will perform a DNS lookup for the `frontend_address` and connect to the addresses found within the DNS record.

Find a docker-compose deployment example at:

- [https://github.com/grafana/tempo/tree/main/example/docker-compose/scalable-single-binary](https://github.com/grafana/tempo/tree/main/example/docker-compose/scalable-single-binary)

## Microservices

In microservices mode, components are deployed in distinct processes.
Scaling is per component, leading to flexibility in scaling.

The configuration associated with each component's deployment specifies a
`target`.  For example, to deploy a `querier`, the configuration would contain
`target: querier`.  A command-line deployment may specify the `-target=querier`
flag. Each of the components referenced in [Architecture]({{< relref
"./architecture" >}}) must be deployed in order to get a working Tempo
instance.

Find a docker-compose deployment example at:

- [https://github.com/grafana/tempo/tree/main/example/docker-compose/distributed](https://github.com/grafana/tempo/tree/main/example/docker-compose/distributed)
