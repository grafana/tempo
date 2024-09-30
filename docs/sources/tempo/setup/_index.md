---
title: Set up Tempo
menuTitle: Set up
description: Learn how to set up a Tempo server or cluster and visualize data.
aliases:
- /docs/tempo/setup
weight: 300
---

# Set up Tempo

To set up Tempo, you need to:

1. Plan your deployment
1. Deploy Tempo
1. Test your installation
1. (Optional) Configure Tempo services

## Plan your deployment

How you choose to deploy Tempo depends upon your tracing needs.
Tempo has two deployment modes: monolithic or microservices.

Read [Plan your deployment]({{< relref "./deployment" >}}) to determine the best method to deploy Tempo.

## Deploy Tempo

Once you have decided how to deploy Tempo, you can install and set up Tempo. For additional samples, refer to the [Example setups]({{< relref "../getting-started/example-demo-app" >}}) topic.

Grafana Tempo is available as a [pre-compiled binary, OS_specific packaging](https://github.com/grafana/tempo/releases), and [Docker image](https://github.com/grafana/tempo/tree/main/example/docker-compose).

The following procedures provide example Tempo deployments that you can use as a starting point:

- [Deploy with Helm]({{< relref "./helm-chart" >}}) (microservices and monolithic)
- [Deploy with Tempo Operator]({{< relref "./operator" >}}) (microservices)
- [Deploy on Linux]({{< relref "./linux" >}}) (monolithic)
- [Deploy on Kubernetes using Tanka]({{< relref "./tanka" >}}) (microservices)

You can also use Docker to deploy Tempo using [the Docker examples](https://github.com/grafana/tempo/tree/main/example/docker-compose).

## Test your installation

Once Tempo is deployed, you can test Tempo by visualizing traces data:

- Using a [test application for a Tempo cluster]({{< relref "./set-up-test-app" >}}) for the Kubernetes with Tanka setup
- Using a [Docker example]({{< relref "./linux" >}}) to test the Linux setup

These visualizations test Kubernetes with Tanka and Linux procedures. They do not check optional configuration you have enabled.

## (Optional) Configure Tempo services

Explore Tempo's features by learning about [available features and configurations]({{< relref "../configuration" >}}).

If you would like to see a simplified, annotated example configuration for Tempo, the [Introduction To MLT](https://github.com/grafana/intro-to-mlt) example repository contains a [configuration](https://github.com/grafana/intro-to-mlt/blob/main/tempo/tempo.yaml) for a monolithic instance.
