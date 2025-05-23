(import 'dashboards/tempo-reads.libsonnet') +
(import 'dashboards/tempo-resources.libsonnet') +
(import 'dashboards/tempo-tenants.libsonnet') +
(import 'dashboards/tempo-writes.libsonnet') +
(import 'dashboards/tempo-block-builder.libsonnet') +
{
  grafanaDashboards+:
    (import 'dashboards/rollout-progress.libsonnet') +
    {
      _config:: $._config,
      'tempo-operational.json': import './dashboards/tempo-operational.json',
      'tempo-backendwork.json': import './dashboards/tempo-backendwork.json',
    },
}
