(import 'common.libsonnet') +
(import 'configmap.libsonnet') +
(import 'config.libsonnet') +
(import 'distributor.libsonnet') +
(import 'generator.libsonnet') +
(import 'frontend.libsonnet') +
(import 'querier.libsonnet') +
(import 'block-builder.libsonnet') +
(import 'live-store.libsonnet') +
(import 'backend-scheduler.libsonnet') +
(import 'backend-worker.libsonnet') +
(import 'vulture.libsonnet') +
(import 'memcached.libsonnet') +
(import 'memberlist.libsonnet') +
(import 'vertical-pod-autoscaler.libsonnet') +
(import 'pod-disruption-budget.libsonnet') +
(import 'rollout-operator/rollout-operator.libsonnet') +

{
  local k = import 'ksonnet-util/kausal.libsonnet',
  namespace:
    k.core.v1.namespace.new($._config.namespace),
}
