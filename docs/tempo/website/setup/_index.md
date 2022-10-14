---
title: Set up a Tempo cluster
menuTitle: Set up a Tempo cluster
description: Learn how to set up a Tempo cluster and visualize data
weight: 150
---

# Set up a Tempo cluster

Tempo is available as a pre-compiled binary, a Docker image, and as common OS-specific packaging.

> **Note:** You can use [Grafana Cloud](https://grafana.com/products/cloud/features/#cloud-traces) to avoid installing, maintaining, and scaling your own instance of Tempo. The free forever plan includes 50GB of free traces. [Create an account to get started](https://grafana.com/auth/sign-up/create-user?pg=docs-enterprise-traces&plcmt=in-text).

## Choose a name for your cluster

No matter which deployment method you choose, you will need to decide what to call your Tempo cluster.
A cluster name must meet the following criteria:

- is 3 to 63 characters long
- contains lowercase letters, numbers, underscores (_), or hyphens (-)
- begins with a letter or number
- ends with a letter or number

## Deploy your cluster

Choose a method to deploy your Tempo cluster:

<!-- - [Deploy on Linux]({{< relref "linux">}}) -->
- [Deploy on Kubernetes using Tanka]({{< relref "tanka">}})

## Test your cluster

Once your cluster is deployed, you can test your cluster by visualizizing the traces data with a simple TNS app. 
Refer to [Set up a test application for Tempo cluster]({{< relref "set-up-test-app" >}}) for instructions.