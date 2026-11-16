// Renders the alerts and rules with a per_cluster_label other than the default, so
// `make check` fails on any that still hard-code `cluster`. See #2000 and #8313.
local mixin = (import 'mixin.libsonnet') + { _config+:: { per_cluster_label: 'k8s_cluster' } };

std.manifestYamlDoc({ alerts: mixin.prometheusAlerts, rules: mixin.prometheusRules })
