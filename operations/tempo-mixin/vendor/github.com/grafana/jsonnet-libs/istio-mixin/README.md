# Istio Mixin

The Istio Mixin is a collection of Grafana dashboards for operating an [Istio](https://istio.io/) service mesh.

The dashboards are based on the [official ones maintained](https://istio.io/latest/docs/tasks/observability/metrics/using-istio-dashboard/) by the Istio community. In addition, the mixin aims to provide [prometheus recording rules](https://prometheus.io/docs/prometheus/latest/configuration/recording_rules/) and [alerting rules](https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/) to operate istio in production.

To use this mixin, install [Tanka](https://tanka.dev/) and [Jsonnet Bundler](https://tanka.dev/install#jsonnet-bundler).

Then you can install the mixin with:

```
jb install github.com/grafana/jsonnet-libs/istio-mixin
```

To use, in your Tanka environment's `main.jsonnet` file:

```jsonnet
local prometheus = (import "prometheus-ksonnet/prometheus-ksonnet.libsonnet");
local istio_mixin = (import "istio-mixin/mixin.libsonnet");

prometheus + istio_mixin {
  _config+:: {
    namespace: "default",
  },
}
```
