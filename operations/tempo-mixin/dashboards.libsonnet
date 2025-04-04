(import 'dashboards/tempo-operational.libsonnet') +
(import 'dashboards/tempo-reads.libsonnet') +
(import 'dashboards/tempo-resources.libsonnet') +
(import 'dashboards/tempo-tenants.libsonnet') +
(import 'dashboards/tempo-writes.libsonnet') +
(import 'dashboards/tempo-block-builder.libsonnet') +
{
  grafanaDashboards+:
    (import 'dashboards/rollout-progress.libsonnet') +

    { _config:: $._config },
}
