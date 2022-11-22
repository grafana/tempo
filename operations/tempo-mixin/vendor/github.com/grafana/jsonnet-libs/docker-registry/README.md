# docker-registry jsonnet library

Jsonnet library for [docker registry](https://docs.docker.com/registry/). 

`withProxy()` provides a quick way to setup a
[mirror](https://docs.docker.com/registry/recipes/mirror/).

## Usage

Install it with jsonnet-bundler:

```console
jb install github.com/grafana/jsonnet-libs/docker-registry
```

Import into your jsonnet:

```jsonnet
// environments/default/main.jsonnet
local registry = import 'github.com/grafana/jsonnet-libs/docker-registry/main.libsonnet';

{
  registry:
    registry.new()
    + registry.withMetrics()
    + registry.withProxy('proxy-credentials-secret')
    + registry.withIngress(
      'registry.example.org',
      tlsSecretName='example-org-wildcard',
      allowlist=['10.0.0.0/8']
    ),
}
```
