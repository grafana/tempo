---
title: Deploy Tempo on Kubernetes
description: Instructions for deploying Tempo on Kubernetes
weight: 400
---

# Deploy Tempo on Kubernetes

Kubernetes manages distributed deployments and provides orchestration for containerized applications.

Kubernetes offers several deployment options for Tempo:

- **Helm charts**: Package manager approach for installation and configuration
- **Tempo Operator**: Kubernetes-native operator for managing Tempo deployments
- **Tanka/Jsonnet**: Configuration management using Jsonnet templating
- **Kubernetes manifests**: Direct YAML configuration files

Choose the method that best fits your Kubernetes workflow and operational requirements.

## Helm

Helm charts are available in the `grafana/helm-charts` repository:

- [monolithic mode](https://github.com/grafana/helm-charts/tree/main/charts/tempo)
- [microservices mode](https://github.com/grafana/helm-charts/tree/main/charts/tempo-distributed) and [`tempo-distributed` chart documentation](/docs/helm-charts/tempo-distributed/next/)

In addition, several Helm chart examples are available in the Tempo repository.

### Kubernetes Tempo Operator

The operator is available in [grafana/tempo-operator](https://github.com/grafana/tempo-operator) repository.
The operator reconciles `TempoStack` resource to deploy and manage Tempo microservices installation.

Refer to the [operator documentation](../kubernetes/operator/) for more details.

### Tanka/Jsonnet

The Jsonnet files that you need to deploy Tempo with Tanka are available here:

- [monolithic mode](https://github.com/grafana/tempo/tree/main/operations/jsonnet/single-binary)
- [microservices mode](https://github.com/grafana/tempo/tree/main/operations/jsonnet/microservices)

Here are a few [examples](https://github.com/grafana/tempo/tree/main/example/tk) that use official Jsonnet files.
They display the full range of configurations available to Tempo.

Refer to [Deploy on Kubernetes with Tanka](/docs/tempo/<TEMPO_VERSION>/set-up-for-tracing/setup-tempo/deploy/kubernetes/tanka/) for an example installation using Tanka.

### Kubernetes manifests

You can find a collection of Kubernetes manifests to deploy Tempo in the
[operations/jsonnet-compiled](https://github.com/grafana/tempo/tree/main/operations/jsonnet-compiled)
folder. These are generated using the Tanka/Jsonnet.
