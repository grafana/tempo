---
title: Set up a Tempo server or cluster
menuTitle: Set up a Tempo server or cluster
description: Learn how to set up a Tempo server or cluster and visualize data
weight: 150
---

# Set up a Tempo server or cluster

Tempo is available as a pre-compiled binary, a Docker image, and as common OS-specific packaging.

You can choose to install Tempo on a single system or as a cluster.

This page highlights these steps; more detailed instructions are available on the procedures for installing Tempo.

## Name your cluster

If you install Tempo in a cluster, you need to decide what to call it.
A cluster name must meet the following criteria:

- is 3 to 63 characters long
- contains lowercase letters, numbers, underscores (_), or hyphens (-)
- begins with a letter or number
- ends with a letter or number

## Deploy Tempo

Choose a method to deploy Tempo:

- [Deploy on Linux]({{< relref "linux">}})
- [Deploy on Kubernetes using Tanka]({{< relref "tanka">}})

## Test your installation

Once Tempo is deployed, you can test your cluster by visualizing the traces data with a simple TNS app.
Refer to [Set up a test application for Tempo cluster]({{< relref "set-up-test-app" >}}) for instructions.

The [Linux]({{< relref "linux">}}) installation provides a verification procedure.
