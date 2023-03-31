---
title: Set up Tempo
menuTitle: Set up Tempo
description: Learn how to set up a Tempo server or cluster and visualize data.
aliases:
- /docs/tempo/setup
weight: 300
---

# Set up

Tempo can deployed in one of three modes:

- monolithic
- scalable monolithic
- microservices

Grafana Tempo is available as a pre-compiled binary, a Docker image, and as common OS-specific packaging.

## Deploy Tempo

How you choose to deploy Tempo depends upon your tracing needs.
Read [Plan your deployment]({{< relref "./deployment" >}}) to determine the best method to deploy Tempo.

THe following procedures provide example Tempo deployments that you can use as a starting point:

- Microservices: [Deploy on Kubernetes using Helm](/docs/helm-charts/tempo-distributed/next/)
- [Deploy on Linux]({{< relref "linux">}})
- [Deploy on Kubernetes using Tanka]({{< relref "tanka">}})

You can also use Docker to deploy Tempo using [the Docker examples](https://github.com/grafana/tempo/tree/main/example/docker-compose).

## Test your installation

Once Tempo is deployed, you can test Tempo by visualizing traces data:

- Using a [test application for a Tempo cluster]({{< relref "set-up-test-app" >}}) for the Kubernetes with Tanka setup
- Using a [Docker example]({{< relref "linux">}}) to test the Linux setup
