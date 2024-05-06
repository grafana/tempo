---
title: Plan your Tempo deployment
menuTitle: Plan your deployment
description: Plan your Grafana Tempo deployment
aliases:
  - /docs/tempo/deployment
  - /docs/tempo/deployment/deployment
  - /docs/tempo/setup/deployment
weight: 200
---

# Plan your Tempo deployment

Tempo can be deployed in _monolithic_ or _microservices_ modes.

The deployment mode is determined by the runtime configuration `target`, or
by using the `-target` flag on the command line. The default target is `all`,
which is the monolithic deployment mode.

{{< admonition type="note" >}}
_Monolithic mode_ was previously called _single binary mode_. Similarly _scalable monolithic mode_ was previously called _scalable single binary mode_. While the documentation has been updated to reflect this change, some URL names and deployment tooling (for example, Helm charts) do not yet reflect this change.
{{% /admonition %}}

## Monolithic mode

Monolithic mode deployment runs all top-level components in a single
process, forming an instance of Tempo. The monolithic mode is the simplest
to deploy, but can not horizontally scale out by increasing the quantity of
components. Refer to [Architecture]({{< relref "../operations/architecture" >}}) for
descriptions of the components.

To enable this mode, `-target=all` is used, which is the default.

Find docker-compose deployment examples at:

- [https://github.com/grafana/tempo/tree/main/example/docker-compose](https://github.com/grafana/tempo/tree/main/example/docker-compose/)

### Scaling monolithic mode

Monolithic mode can be horizontally scaled out.
This scalable monolithic mode is similar to the monolithic mode in that all components are run within one process.
Horizontal scale out is achieved by instantiating more than one process, with each having `-target` set to `scalable-single-binary`.

This mode offers some flexibility of scaling without the configuration complexity of the full
microservices deployment.

Each of the `queriers` perform a DNS lookup for the `frontend_address` and connect to the addresses found within the DNS record.

Find a docker-compose deployment example at:

- [https://github.com/grafana/tempo/tree/main/example/docker-compose/scalable-single-binary](https://github.com/grafana/tempo/tree/main/example/docker-compose/scalable-single-binary)

## Microservices mode

In microservices mode, components are deployed in distinct processes.
Scaling is per component, which allows for greater flexibility in scaling and more
granular failure domains. This is the preferred method for a production
deployment, but it is also the most complex.

The configuration associated with each component's deployment specifies a
`target`. For example, to deploy a `querier`, the configuration would contain
`target: querier`. A command-line deployment may specify the `-target=querier`
flag. Each of the components referenced in [Architecture]({{< relref
"../operations/architecture" >}}) must be deployed in order to get a working Tempo
instance.

Find a docker-compose deployment example at:

- [https://github.com/grafana/tempo/tree/main/example/docker-compose/distributed](https://github.com/grafana/tempo/tree/main/example/docker-compose/distributed)

## Tools used to deploy Tempo

Tempo can be easily deployed through a number of tools, including Helm, Tanka, Kubernetes, and Docker.

{{< admonition type="note" >}}
The Tanka and Helm examples are equivalent.
They are both provided for people who prefer different configuration mechanisms.
{{% /admonition %}}

### Helm

Helm charts are available in the `grafana/helm-charts` repository:

- [monolithic mode](https://github.com/grafana/helm-charts/tree/main/charts/tempo)
- [microservices mode](https://github.com/grafana/helm-charts/tree/main/charts/tempo-distributed) and [`tempo-distributed` chart documentation](/docs/helm-charts/tempo-distributed/next/)

In addition, several Helm chart examples are available in the Tempo repository.

### Kubernetes Tempo Operator

The operator is available in [grafana/tempo-operator](https://github.com/grafana/tempo-operator) repository.
The operator reconciles `TempoStack` resource to deploy and manage Tempo microservices installation.

Refer to the [operator documentation]({{< relref "./operator" >}}) for more details.

### Tanka/Jsonnet

The Jsonnet files that you need to deploy Tempo with Tanka are available here:

- [monolithic mode](https://github.com/grafana/tempo/tree/main/operations/jsonnet/single-binary)
- [microservices mode](https://github.com/grafana/tempo/tree/main/operations/jsonnet/microservices)

Here are a few [examples](https://github.com/grafana/tempo/tree/main/example/tk) that use official Jsonnet files.
They display the full range of configurations available to Tempo.


### Kubernetes manifests

You can find a collection of Kubernetes manifests to deploy Tempo in the
[operations/jsonnet-compiled](https://github.com/grafana/tempo/tree/main/operations/jsonnet-compiled)
folder. These are generated using the Tanka/Jsonnet.
