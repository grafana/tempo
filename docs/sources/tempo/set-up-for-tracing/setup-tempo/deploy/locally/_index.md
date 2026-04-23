---
title: Deploy Tempo locally
description: Instructions for deploying Tempo locally using Docker Compose or on Linux
weight: 400
---

# Deploy Tempo locally

This section provides instructions for deploying Tempo locally using Docker Compose or directly on a Linux host.

You can download the latest version from the [Tempo Releases page](https://github.com/grafana/tempo/releases).

{{< section withDescriptions="true">}}

{{< admonition type="note" >}}
Grafana Tempo doesn't come with any included authentication layer. You must run an authenticating reverse proxy in front of your services to prevent unauthorized access to Tempo (for example, nginx). [Manage authentication](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/authentication/) for more details
{{< /admonition >}}
