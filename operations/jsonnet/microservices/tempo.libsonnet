(import 'common.libsonnet') +
(import 'configmap.libsonnet') +
(import 'config.libsonnet') +
(import 'compactor.libsonnet') +
(import 'distributor.libsonnet') +
(import 'ingester.libsonnet') +
(import 'generator.libsonnet') +
(import 'frontend.libsonnet') +
(import 'querier.libsonnet') +
(import 'block-builder.libsonnet') +
(import 'backend-scheduler.libsonnet') +
(import 'backend-worker.libsonnet') +
(import 'vulture.libsonnet') +
(import 'memcached.libsonnet') +
(import 'multi-zone.libsonnet') +
(import 'memberlist.libsonnet') +
(import 'rollout-operator.libsonnet') +

{
  local k = import 'ksonnet-util/kausal.libsonnet',
  namespace:
    k.core.v1.namespace.new($._config.namespace),
}
