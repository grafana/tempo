local example = import 'example.jsonnet';
local prom = import 'prom.libsonnet';

example {
  prometheusAlerts+: prom.v1.patchRule('prometheus_metamon', 'PrometheusDown', { 'for': '10m' }),
}
