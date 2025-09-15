---
title: Deploy Tempo
menuTitle: Deploy your Tempo instance
description: Deploy Grafana Tempo for your tracing needs.
weight: 300
aliases:
  - ../../deployment/ # /docs/tempo/next/deployment/
  - ../../deployment/deployment/ # /docs/tempo/next/deployment/deployment/
  - ../../setup/deployment/ # /docs/tempo/next/setup/deployment/
---

# Deploy Tempo

Tempo can be easily deployed through a number of tools, including Helm, Tanka, Kubernetes, and Docker.

The following procedures provide example Tempo deployments that you can use as a starting point.

Tempo can be deployed in a number of ways, depending on your needs and environment. You can deploy Tempo in a monolithic mode or in a microservices mode.

You can also use Docker to deploy Tempo using [the Docker examples](https://github.com/grafana/tempo/tree/main/example/docker-compose).

## Deploy locally

Monolithic mode (single binary) is commonly used for a local installation, testing, or small-scale deployments.
This mode can be deployed using a pre-compiled binary, OS-specific packaging, or Docker image.
While it's possible to deploy monolithic mode in a Kubernetes cluster, it is not recommended for production use.

You can deploy Tempo locally using the monolithic mode. You can use using the [Docker examples](https://github.com/grafana/tempo/tree/main/example/docker-compose) or you can use the [Linux example](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/deploy/locally/linux/) (monolithic) to deploy Tempo on a Linux host.

For more information, refer [Deploy Tempo locally](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/deploy/locally/).

## Deploy using Kubernetes

Kubernetes deployment examples:

- [Deploy with Helm](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/deploy/kubernetes/helm-chart/) (microservices and monolithic)
- [Deploy with Tempo Operator](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/deploy/kubernetes/operator/) (microservices)
- [Deploy on Kubernetes using Tanka](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/deploy/kubernetes/tanka/) (microservices)

{{< admonition type="note" >}}
The Tanka and Helm examples are equivalent.
They are both provided for people who prefer different configuration mechanisms.
{{< /admonition >}}

For more information, refer to [Deploy Tempo on Kubernetes](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/deploy/kubernetes/).
