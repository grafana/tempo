---
title: Deploy Tempo
menuTitle: Deploy your Tempo instance
description: Deploy Grafana Tempo for your tracing needs.
aliases:
  - /docs/tempo/deployment
  - /docs/tempo/deployment/deployment
  - /docs/tempo/setup/deployment
weight: 300
---

# Deploy Tempo

Tempo can be easily deployed through a number of tools, including Helm, Tanka, Kubernetes, and Docker.

The following procedures provide example Tempo deployments that you can use as a starting point.

Local deployment examples:
- [Deploy on Linux](linux/) (monolithic)

You can also use Docker to deploy Tempo using [the Docker examples](https://github.com/grafana/tempo/tree/main/example/docker-compose).

## Deploy locally

You can deploy Tempo locally using the monolithic mode. You can use using the [Docker examples](https://github.com/grafana/tempo/tree/main/example/docker-compose).

For more information, refer [Deploy Tempo locally](../deploy/locally/).

## Deploy using Kubernetes

Kubernetes deployment examples:
- [Deploy with Helm](helm-chart/) (microservices and monolithic)
- [Deploy with Tempo Operator](operator/) (microservices)
- [Deploy on Kubernetes using Tanka](tanka/) (microservices)

{{< admonition type="note" >}}
The Tanka and Helm examples are equivalent.
They are both provided for people who prefer different configuration mechanisms.
{{< /admonition >}}

For more information, refer to [Deploy Tempo on Kubernetes](../deploy/kubernetes/).


