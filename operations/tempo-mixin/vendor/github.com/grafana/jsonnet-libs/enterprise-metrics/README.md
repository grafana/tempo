# Grafana Enterprise Metrics Jsonnet library

> **Note:** This library is **DEPRECATED** and will be removed in the future. If you are a Grafana Labs customer
> making use of this library, please contact support for help migrating to our [helm chart](https://grafana.com/docs/mimir/v2.1.x/operators-guide/deploying-grafana-mimir/).



A [Jsonnet](https://jsonnet.org/) library that can be used to configure and deploy Grafana Enterprise Metrics to Kubernetes.

## Getting started

[Jsonnet bundler](https://github.com/jsonnet-bundler/jsonnet-bundler) is used to manage Jsonnet dependencies.
Dependencies will be installed into the `/vendor` directory.
To install the library and its dependencies using Jsonnet bundler:

```console
$ jb install github.com/grafana/jsonnet-libs/enterprise-metrics
```

`k.libsonnet` is a Jsonnet library for Kubernetes. Most other Jsonnet libraries for Kubernetes expect this file to exist in your JSONNET_PATH.

To install `k.libsonnet` for Kubernetes 1.18:

1. Use `jb` to vendor the `k8s-libsonnet` Kubernetes library
```console
$ jb install github.com/jsonnet-libs/k8s-libsonnet/1.18@main
```

2. Ensure the `lib` directory exists
```console
$ mkdir -p lib
```

3. Import the `k8s-libsonnet` Kubernetes library by putting the following Jsonnet into a file called `lib/k.libsonnet`
```jsonnet
(import 'github.com/jsonnet-libs/k8s-libsonnet/1.18/main.libsonnet')
```

## Development

In order to reflect changes made to the library also in the documentation (`docs/README.md`), one
needs to run the `docs` target:

```console
$ make docs
```

### Unit tests

To run the unit tests:

```console
$ make test
```

### QA testing

To build all the manifests:

```console
$ make eval
```

To deploy a minimal GEM to a local k3d cluster:

```console
$ make local-test
```
