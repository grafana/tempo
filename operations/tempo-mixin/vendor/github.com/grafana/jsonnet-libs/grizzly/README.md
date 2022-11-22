# Grizzly Library

This library provides utilities for [Grizzly](https://github.com/grafana/grizzly),
a utility for managing observability resources on Grafana and hosted Prometheus
installations.

## Kubernetes Style Observability Objects
It provides functions to render Kubernetes style objects from Monitoring Mixins.

If deploying dashboards structured to be consumed by [prometheus-ksonnet](https://github.com/grafana/jsonnet-libs/prometheus-ksonnet)
or associated libraries, where other resources are using Tanka, place a file
named `grr.jsonnet` next to your `main.jsonnet`, and give it this content:

```
local main = (import 'main.jsonnet');
local grizzly = (import 'grizzly/grizzly.libsonnet');

grizzly.fromPrometheusKsonnet(main)
```

Then, you can invoke this from Grizzly like so:

`grr show -r environments/<my-environment>/grr.libsonnet`
