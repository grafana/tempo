// The grafana-databases-lib instance shared by Tempo's dashboards, configured from $._config.
// It is merged into dashboard-utils.libsonnet, so every dashboard can use $._databases. Each area
// of the library (autoscaling, ...) adds its own config mixin below.
local dblib = import 'grafana-databases-lib/main.libsonnet';

{
  _databases::
    dblib.new({
      selector: $.namespaceMatcher(),
      clusterLabel: $._config.per_cluster_label,
      groupBy: $._config.cluster_selectors,
    })
    // Autoscaling panels and rows, one set per component defined in $._config.autoscaling.
    + dblib.autoscaling.withConfigMixin({
      hpaPrefix: $._config.autoscaling_hpa_prefix,
      components: {
        [name]: {
          hpaName: $._config.autoscaling[name].hpa_name,
          keda: $._config.autoscaling[name].keda_triggers,
        }
        for name in std.objectFields($._config.autoscaling)
      },
    }),
}
