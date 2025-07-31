---
title: Set up Tempo
menuTitle: Set up Tempo
description: Learn how to set up a Tempo server or cluster and visualize data.
aliases:
  - ../setup/ # /docs/tempo/next/setup/
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

Refer to [Plan your deployment](deployment/) to determine the best method to deploy Tempo.

## Deploy Tempo

Once you have decided the best method, you can deploy and set up Tempo.

Grafana Tempo is available as a [pre-compiled binary, OS-specific packaging](https://github.com/grafana/tempo/releases), and [Docker image](https://github.com/grafana/tempo/tree/main/example/docker-compose).

Refer to [Deploy Tempo](deploy/) for instructions for deploying using Kubernetes and deploying locally.

## Test your installation

Once Tempo is deployed, you can test Tempo by visualizing traces data:

- Using a [test application for a Tempo cluster](set-up-test-app/) for the Kubernetes with Tanka setup
- Using a [Docker example](linux/) to test the Linux setup

These visualizations test Kubernetes with Tanka and Linux procedures.
They don't check optional configuration you have enabled.

## (Optional) Configure Tempo services

Explore Tempo's features by learning about [available features and configurations](../configuration/).

If you would like to see a simplified, annotated example configuration for Tempo, the [Introduction To MLT](https://github.com/grafana/intro-to-mlt) example repository contains a [configuration](https://github.com/grafana/intro-to-mlt/blob/main/tempo/tempo.yaml) for a monolithic instance.
