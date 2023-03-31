---
title: Deploying Tempo
aliases:
  - /docs/tempo/latest/deployment
  - /docs/tempo/latest/deployment/deployment
weight: 30
---

# Deploying Tempo

Tempo can be easily deployed through a number of tools as explained in this document.

> **Note**: The Tanka and Helm examples are equivalent.
> They are both provided for people who prefer different configuration mechanisms.

## Tanka/Jsonnet

The Jsonnet files that you need to deploy Tempo with Tanka are available here:

- [monolithic mode](https://github.com/grafana/tempo/tree/main/operations/jsonnet/single-binary)
- [microservices mode](https://github.com/grafana/tempo/tree/main/operations/jsonnet/microservices)

Here are a few [examples](https://github.com/grafana/tempo/tree/main/example/tk) that use official Jsonnet files.
They display the full range of configurations available to Tempo.

## Helm

Helm charts are available in the grafana/helm-charts repo:

- [monolithic mode](https://github.com/grafana/helm-charts/tree/main/charts/tempo)
- [microservices mode](https://github.com/grafana/helm-charts/tree/main/charts/tempo-distributed) and [`tempo-distributed` documentation](/docs/helm-charts/tempo-distributed/next/)

## Kubernetes manifests

You can find a collection of Kubernetes manifests to deploy Tempo in the
[operations/jsonnet-compiled](https://github.com/grafana/tempo/tree/main/operations/jsonnet-compiled)
folder.  These are generated using the Tanka / Jsonnet.

## Deployment scenarios

Tempo can be deployed in one of three modes:

- monolithic
- scalable monolithic
- microservices

Which mode is deployed is determined by the runtime configuration `target`, or
by using the `-target` flag on the command line. The default target is `all`,
which is the monolithic deployment mode.

> **Note:** _Monolithic mode_ was previously called _single binary mode_. Similarly _scalable monolithic mode_ was previously called _scalable single binary mode_. While the documentation has been updated to reflect this change, some URL names and deployment tooling (e.g. Helm charts) do not yet reflect this change.

### Monolithic

Monolithic mode deployment runs all top-level components in a single
process, forming an instance of Tempo.  The monolithic mode is the simplest
to deploy, but can not horizontally scale out by increasing the quantity of
components.  Refer to [Architecture]({{< relref "./architecture" >}}) for
descriptions of the components.

To enable this mode, `-target=all` is used, which is the default.

Find docker-compose deployment examples at:

- [https://github.com/grafana/tempo/tree/main/example/docker-compose/local](https://github.com/grafana/tempo/tree/main/example/docker-compose/local)
- [https://github.com/grafana/tempo/tree/main/example/docker-compose/s3](https://github.com/grafana/tempo/tree/main/example/docker-compose/s3)

### Scalable monolithic

Scalable monolithic mode is similar to the monolithic mode in
that all components are run within one process. Horizontal scale out is
achieved by instantiating more than one process, with each having `-target` set to `scalable-single-binary`.

This mode offers some
flexibility of scaling without the configuration complexity of the full
microservices deployment.

Each of the `queriers` will perform a DNS lookup for the `frontend_address` and
connect to the addresses found within the DNS record.

Find a docker-compose deployment example at:

- [https://github.com/grafana/tempo/tree/main/example/docker-compose/scalable-single-binary](https://github.com/grafana/tempo/tree/main/example/docker-compose/scalable-single-binary)

### Microservices

In microservices mode, components are deployed in distinct processes.  Scaling
is per component, which allows for greater flexibility in scaling and more
granular failure domains. This is the preferred method for a production
deployment, but it is also the most complex

The configuration associated with each component's deployment specifies a
`target`.  For example, to deploy a `querier`, the configuration would contain
`target: querier`.  A command-line deployment may specify the `-target=querier`
flag. Each of the components referenced in [Architecture]({{< relref
"./architecture" >}}) must be deployed in order to get a working Tempo
instance.

Find a docker-compose deployment example at:

- [https://github.com/grafana/tempo/tree/main/example/docker-compose/distributed](https://github.com/grafana/tempo/tree/main/example/docker-compose/distributed)
