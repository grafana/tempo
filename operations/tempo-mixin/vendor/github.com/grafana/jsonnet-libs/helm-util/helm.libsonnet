std.trace(
  'Deprecated, please use `github.com/grafana/jsonnet-libs/tanka-util/` instead. (2020-12-11)',
  (import 'github.com/grafana/jsonnet-libs/tanka-util/helm.libsonnet')
  + (import 'github.com/grafana/jsonnet-libs/tanka-util/k8s.libsonnet')
)
