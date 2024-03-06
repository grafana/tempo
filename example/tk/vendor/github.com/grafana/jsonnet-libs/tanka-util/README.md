---
permalink: /
---

# tanka_util

```jsonnet
local tanka_util = import "github.com/grafana/jsonnet-libs/tanka-util/main.libsonnet"
```

Package `tanka_util` provides jsonnet tooling that works well with
[Grafana Tanka](https://tanka.dev) features. This package implements
[Helm](https://tanka.dev/helm) and [Kustomize](https://tanka.dev/helm) 
support for Grafana Tanka.

### Usage

> **Warning:** [Functionality required](#internals) by this library is still
> experimental and may break.

The [`helm.template`](#fn-helmtemplate) function converts a Helm Chart into a
Jsonnet object to be consumed by tools like Tanka. Similarly the
[`kustomize.build`](#fn-kustomizebuild) function expands Kustomizations.

Helm Charts are required to be available on the local file system and are
resolved relative to the file that calls `helm.template`.

Kustomizations are also resolved relative to the file that calls
`kustomize.build`. 

```jsonnet
local tanka = import 'github.com/grafana/jsonnet-libs/tanka-util/main.libsonnet';
local helm = tanka.helm.new(std.thisFile);
local kustomize = tanka.kustomize.new(std.thisFile);

{
  // render the Grafana Chart, set namespace to "test"
  grafana: helm.template('grafana', './charts/grafana', {
    values: {
      persistence: { enabled: true },
      plugins: ['grafana-clock-panel'],
    },
    namespace: 'test',
  }),

  // render the Prometheus Kustomize
  // then entrypoint for `kustomize build` will be ./base/prometheus/kustomization.yaml
  prometheus: kustomize.build('./base/prometheus'),
}

```

For more information on that see https://tanka.dev/helm

### Internals

The functionality of `helm.template` is based on the `helm template` command.
Because Jsonnet does not support executing arbitrary command for [good
reasons](https://jsonnet.org/ref/language.html#independence-from-the-environment-hermeticity),
a different way was required.

To work around this, [Tanka](https://tanka.dev) instead binds special
functionality into Jsonnet that provides `helm template`.

This however means this library and all libraries using this library are not
compatible with `google/go-jsonnet` or `google/jsonnet`.

Kustomize is build so that each kustomization can pull another kustomization
from the internet. Due to this feature it is not feasible to ensure hermetic and
reprodicible kustomize builds from within Tanka. Beware of that when using the
Kustomize functionality.


## Index

* [`obj environment`](#obj-environment)
  * [`fn new(name, namespace, apiserver)`](#fn-environmentnew)
  * [`fn withApiServer(apiserver)`](#fn-environmentwithapiserver)
  * [`fn withApplyStrategy(applyStrategy)`](#fn-environmentwithapplystrategy)
  * [`fn withData(data)`](#fn-environmentwithdata)
  * [`fn withDataMixin(data)`](#fn-environmentwithdatamixin)
  * [`fn withInjectLabels(bool)`](#fn-environmentwithinjectlabels)
  * [`fn withLabels(labels)`](#fn-environmentwithlabels)
  * [`fn withLabelsMixin(labels)`](#fn-environmentwithlabelsmixin)
  * [`fn withName(name)`](#fn-environmentwithname)
  * [`fn withNamespace(namespace)`](#fn-environmentwithnamespace)
  * [`fn withResourceDefaults(labels)`](#fn-environmentwithresourcedefaults)
  * [`fn withResourceDefaultsMixin(labels)`](#fn-environmentwithresourcedefaultsmixin)
* [`obj helm`](#obj-helm)
  * [`fn new(calledFrom)`](#fn-helmnew)
  * [`fn template(name, chart, conf)`](#fn-helmtemplate)
* [`obj k8s`](#obj-k8s)
  * [`fn patchKubernetesObjects(object, patch)`](#fn-k8spatchkubernetesobjects)
  * [`fn patchLabels(object, labels)`](#fn-k8spatchlabels)
* [`obj kustomize`](#obj-kustomize)
  * [`fn new(calledFrom)`](#fn-kustomizenew)
  * [`fn build(path, conf)`](#fn-kustomizebuild)

## Fields

## obj environment

`environment` provides a base to create an [inline Tanka
environment](https://tanka.dev/inline-environments#inline-environments).


### fn environment.new

```ts
new(name, namespace, apiserver)
```

`new` initiates an [inline Tanka environment](https://tanka.dev/inline-environments#inline-environments)


### fn environment.withApiServer

```ts
withApiServer(apiserver)
```

`withApiServer` sets the Kubernetes cluster this environment should apply to.
Must be the full URL, e.g. https://cluster.fqdn:6443


### fn environment.withApplyStrategy

```ts
withApplyStrategy(applyStrategy)
```

`withApplyStrategy` sets the Kubernetes apply strategy used for this environment.
Must be `client` or `server`


### fn environment.withData

```ts
withData(data)
```

`withData` adds the actual Kubernetes resources to the inline environment.

### fn environment.withDataMixin

```ts
withDataMixin(data)
```

`withDataMixin` adds the actual Kubernetes resources to the inline environment.
*Note:* This function appends passed data to existing values


### fn environment.withInjectLabels

```ts
withInjectLabels(bool)
```

`withInjectLabels` adds a "tanka.dev/environment" label to each created resource.
Required for [garbage collection](https://tanka.dev/garbage-collection).


### fn environment.withLabels

```ts
withLabels(labels)
```

`withLabels` adds arbitrary key:value labels.

### fn environment.withLabelsMixin

```ts
withLabelsMixin(labels)
```

`withLabelsMixin` adds arbitrary key:value labels.
*Note:* This function appends passed data to existing values


### fn environment.withName

```ts
withName(name)
```

`withName` sets the environment `name`.

### fn environment.withNamespace

```ts
withNamespace(namespace)
```

`withNamespace` sets the default namespace for objects that don't explicitely specify one.

### fn environment.withResourceDefaults

```ts
withResourceDefaults(labels)
```

`withResourceDefaults` sets defaults for all resources in this environment.

### fn environment.withResourceDefaultsMixin

```ts
withResourceDefaultsMixin(labels)
```

`withResourceDefaultsMixin` sets defaults for all resources in this environment.
*Note:* This function appends passed data to existing values


## obj helm

`helm` allows the user to consume Helm Charts as plain Jsonnet resources.
This implements [Helm support](https://tanka.dev/helm) for Grafana Tanka.


### fn helm.new

```ts
new(calledFrom)
```

`new` initiates the `helm` object. It must be called before any `helm.template` call:
 > ```jsonnet
 > // std.thisFile required to correctly resolve local Helm Charts
 > helm.new(std.thisFile)
 > ```


### fn helm.template

```ts
template(name, chart, conf)
```

`template` expands the Helm Chart to its underlying resources and returns them in an `Object`,
so they can be consumed and modified from within Jsonnet.

This functionality requires Helmraiser support in Jsonnet (e.g. using Grafana Tanka) and also
the `helm` binary installed on your `$PATH`.


## obj k8s

`k8s` provides common utils to modify Kubernetes objects.


### fn k8s.patchKubernetesObjects

```ts
patchKubernetesObjects(object, patch)
```

`patchKubernetesObjects` applies `patch` to all Kubernetes objects it finds in `object`.

### fn k8s.patchLabels

```ts
patchLabels(object, labels)
```

`patchLabels` finds all Kubernetes objects and adds labels to them.

## obj kustomize

`kustomize` allows the user to expand Kustomize manifests into plain Jsonnet resources.
This implements [Kustomize support](https://tanka.dev/kustomize) for Grafana Tanka.


### fn kustomize.new

```ts
new(calledFrom)
```

`new` initiates the `kustomize` object. It must be called before any `kustomize.build` call:
 > ```jsonnet
 > // std.thisFile required to correctly resolve local Kustomize objects
 > kustomize.new(std.thisFile)
 > ```


### fn kustomize.build

```ts
build(path, conf)
```

`build` expands the Kustomize object to its underlying resources and returns them in an `Object`,
so they can be consumed and modified from within Jsonnet.

This functionality requires Kustomize support in Jsonnet (e.g. using Grafana Tanka) and also
the `kustomize` binary installed on your `$PATH`.

`path` is relative to the file calling this function.
