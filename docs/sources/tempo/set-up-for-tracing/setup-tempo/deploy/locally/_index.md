---
title: Deploy Tempo locally
description: Instructions for deploying Tempo locally
weight: 400
---

# Deploy Tempo locally

You can deploy Tempo This section provides instructions for deploying Tempo locally using Tanka and Jsonnet or using Docker Compose.

{{< section withDescriptions="true">}}

{{< admonition type="note" >}}
Grafana Tempo does not come with any included authentication layer. You must run an authenticating reverse proxy in front of your services to prevent unauthorized access to Tempo (for example, nginx). [Manage authentication](https://grafana.com/docs/tempo/<TEMPO_VERSION>/operations/authentication/) for more details
{{< /admonition >}}

