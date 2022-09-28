---
title: Set up a GET cluster
weight: 300
---

# Set up a Grafana Enterprise Traces cluster

Grafana Enterprise Traces (GET) is available as a pre-compiled binary, a Docker image, and as common OS-specific packaging. For a list of available download options, refer to [Downloads](../downloads).

> **Note:** You can use [Grafana Cloud](https://grafana.com/products/cloud/features/#cloud-traces) to avoid installing, maintaining, and scaling your own instance of GET. The free forever plan includes 50GB of free traces. [Create an account to get started](https://grafana.com/auth/sign-up/create-user?pg=docs-enterprise-traces&plcmt=in-text).

## Get a GET license

A valid Grafana Enterprise Traces license token is required to run GET's many added features. GET will run with the functionality of an open-source Tempo binary when no license token is available.

To get a license, please [contact a Grafana Labs representative](https://grafana.com/contact?about=grafana-enterprise-logs&amp;pg=prod-gme&amp;plcmt=hero-btn-1).

### Choose a name for your GET cluster

GET licenses are issued on a per-cluster basis. Each cluster of GET that you plan to deploy requires a unique license. When requesting a GET license, you'll be asked to provide a unique cluster name with which to associate the license.

A cluster name must meet the following criteria:

- is 3 to 63 characters long
- contains lowercase letters, numbers, underscores (_), or hyphens (-)
- begins with a letter or number
- ends with a letter or number

### Download your GET license

Your GET license can be downloaded as follows:

1. From [https://grafana.com](https://grafana.com), select **Login**.
1. From the left-hand menu, select **Licenses** to download the license token.

## Deploy your GET cluster

After you have a Grafana GET license with an associated cluster name, choose a method to deploy your GET cluster:

- [Deploy on Linux](linux)
- [Deploy on Kubernetes with Tanka](tanka)
