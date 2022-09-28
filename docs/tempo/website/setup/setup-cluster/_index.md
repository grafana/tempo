---
title: Set up a GET cluster
weight: 300
---

# Set up a Tempo cluster

Tempo is available as a pre-compiled binary, a Docker image, and as common OS-specific packaging. 

> **Note:** You can use [Grafana Cloud](https://grafana.com/products/cloud/features/#cloud-traces) to avoid installing, maintaining, and scaling your own instance of GET. The free forever plan includes 50GB of free traces. [Create an account to get started](https://grafana.com/auth/sign-up/create-user?pg=docs-enterprise-traces&plcmt=in-text).


### Choose a name for your cluster


A cluster name must meet the following criteria:

- is 3 to 63 characters long
- contains lowercase letters, numbers, underscores (_), or hyphens (-)
- begins with a letter or number
- ends with a letter or number

## Deploy your cluster

Choose a method to deploy your Tempo cluster:

- [Deploy on Linux](linux)
- [Deploy on Kubernetes with Tanka](tanka)
