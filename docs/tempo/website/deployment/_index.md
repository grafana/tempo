---
title: Deployment
weight: 250
---

# Deploying Tempo

Tempo can be easily deployed through a number of tools as explained in this document.

> Note: The Tanka and Helm examples are equivalent.
> They are both provided for people who prefer different configuration mechanisms.

## Tanka / Jsonnet

The jsonnet files required to deploy Tempo with Tanka are available here:
- [single binary](https://github.com/grafana/tempo/tree/main/operations/jsonnet/single-binary)
- [microservices](https://github.com/grafana/tempo/tree/main/operations/jsonnet/microservices)

A few [examples](https://github.com/grafana/tempo/tree/main/example/tk) have been provided that use official jsonnet files.
These show off the full range of configuration available to Tempo.

## Helm

Helm charts are available in the grafana/helm-charts repo:

- [single binary](https://github.com/grafana/helm-charts/tree/main/charts/tempo)
- [microservices](https://github.com/grafana/helm-charts/tree/main/charts/tempo-distributed)

