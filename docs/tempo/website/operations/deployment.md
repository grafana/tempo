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

You can find a collection of Kubernetes manifests to deploy Tempo in the [operations/kube-manifests](https://github.com/grafana/tempo/tree/main/operations/kube-manifests) folder.
These are generated using the Tanka / Jsonnet.

# Deployment Scenarios

Tempo can be deployed in three modes.

* Single Binary
* Scalable Single Binary
* Microservices

These modes are controlled by the runtime configuration `target`, or using the CLI flag `-target`.  The default target is `all`, which we refer to as "Single Binary".

## Single Binary

This mode is the simplest to get started, in which all of the top-level components, referenced in the [Architecture]({{< relref "./architecture" >}}) section, are run within the same instance of Tempo.

A couple docker-compose examples can be found [here](https://github.com/grafana/tempo/tree/main/example/docker-compose/local) and [here](https://github.com/grafana/tempo/tree/main/example/docker-compose/s3).

## Scalable Single Binary

This modr allows the simplicity of the Single Binary mode, but utilizes the clustering features to allow for horizontal scalability.

A `kvstore` must be used.  For example, `memberlist` is configured below to illustrate the setup

```yaml
target: scalable-single-binary
ingester:
  lifecycler:
    ring:
      kvstore:
        store: memberlist
```

Additionally, the `queriers` must know the DNS name that will contain the addresses of all other instances.

```yaml
querier:
  frontend_worker:
    frontend_address: tempo.lab.example.com:9095
```

Each of the `queriers` will perform a DNS lookup for the `frontend_address` and connect to addresses found within the DNS record.

An example using docker-compose can be found [here](https://github.com/grafana/tempo/tree/main/example/docker-compose/scalable-single-binary)

## Microservices

In this mode, each component is deployed in only one instance.  For example, to deploy a `querier`, the Tempo configuration would either contain `target: querier` or the binary is to be executed with `-target=querier`.  Each of the comonents referenced in the [Architecture]({{< relref "./architecture" >}}) must be deployed in order to get a working Tempo environment.

An example using docker-compose can be found [here](https://github.com/grafana/tempo/tree/main/example/docker-compose/distributed).