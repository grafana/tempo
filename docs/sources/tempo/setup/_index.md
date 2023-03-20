---
title: Set up a Tempo server or cluster
menuTitle: Set up a Tempo server or cluster
description: Learn how to set up a Tempo server or cluster and visualize data
weight: 150
---

# Set up a Tempo server or cluster

Tempo is available as a pre-compiled binary, a Docker image, and as common OS-specific packaging.

This section describes how to set up Tempo on a single Linux node or as a cluster using Kubernetes and Tanka. Tempo can also be set up as a distributed set of services.

This page highlights these steps; more detailed instructions are available on the procedures for installing Tempo.

## Deploy Tempo

Choose a method to deploy Tempo:

- [Deploy on Kubernetes using Helm](/docs/helm-charts/tempo-distributed/next/)
- [Deploy on Linux]({{< relref "linux">}})
- [Deploy on Kubernetes using Tanka]({{< relref "tanka">}})

You can also use Docker to deploy Tempo using [the Docker examples](https://github.com/grafana/tempo/tree/main/example/docker-compose).

## Test your installation

Once Tempo is deployed, you can test Tempo by visualizing traces data:

- Using a [test application for a Tempo cluster]({{< relref "set-up-test-app" >}}) for the Kubernetes with Tanka setup
- Using a [Docker example]({{< relref "linux">}}) to test the Linux setup
