(import 'common.libsonnet') +
(import 'configmap.libsonnet') +
(import 'config.libsonnet') +
(import 'compactor.libsonnet') +
(import 'distributor.libsonnet') +
(import 'ingester.libsonnet') +
(import 'frontend.libsonnet') +
(import 'querier.libsonnet') +
(import 'vulture.libsonnet') +
(import 'memcached.libsonnet') +
{
  local k = import 'ksonnet-util/kausal.libsonnet',
  namespace:
    k.core.v1.namespace.new($._config.namespace),
}
