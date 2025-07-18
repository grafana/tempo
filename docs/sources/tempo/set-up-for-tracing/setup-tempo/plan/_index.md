---
title: Plan your Tempo deployment
menuTitle: Plan your deployment
description: Plan your Grafana Tempo deployment
aliases:
  - /docs/tempo/deployment
  - /docs/tempo/deployment/deployment
  - /docs/tempo/setup/deployment
weight: 200
---

# Plan your Tempo deployment

When you plan your Tempo deployment, you need to:

* Identify your Tempo use case.
* Consider the amount of tracing data you'll deal with.
* Determine the mode you want to use for your Tempo use case.

## Identify your use case

What capabilities do you need?
* Basic installation
* Horizontally scalable?
* Multi-tenancy?
* Data storage?

## Estimate the tracing data volume

Refer to [Size your cluster](../size/) to estimate the amount of tracing data you need to store and query.

## Determine the deployment mode

Tempo can be deployed in _monolithic_ or _microservices_ modes.
Evaluate the deployment mode that best fits your use case.

Refer to [Deployment modes](deployment-modes/) for more information about the differences between the two modes.