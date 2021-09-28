local memcached = import 'memcached/memcached.libsonnet';

memcached {
  memcached+:: {
    cpu_limits:: null,
    connection_limit: $._config.memcached.connection_limit,
    memory_limit_mb: $._config.memcached.memory_limit_mb,

    deployment: {},

    local k = import 'ksonnet-util/kausal.libsonnet',
    local statefulSet = k.apps.v1.statefulSet,

    statefulSet:
      statefulSet.new(self.name, $._config.memcached.replicas, [
        self.memcached_container,
        self.memcached_exporter,
      ], []) +
      statefulSet.mixin.spec.withServiceName(self.name) +
      k.util.antiAffinityStatefulSet,

    local service = k.core.v1.service,

    service:
      k.util.serviceFor(self.statefulSet) +
      service.mixin.spec.withClusterIp('None'),
  },

  // Dedicated memcached instance used to cache query results.
  memcached_all: $.memcached {
    name: 'memcached',
    max_item_size: '5m',
  },
}
