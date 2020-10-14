local memcached = import 'memcached/memcached.libsonnet';

memcached {
  memcached+:: {
    cpu_limits:: null,

    deployment: {},

    local statefulSet = $.apps.v1.statefulSet,

    statefulSet:
      statefulSet.new(self.name, 3, [
        self.memcached_container,
        self.memcached_exporter,
      ], []) +
      statefulSet.mixin.spec.withServiceName(self.name) +
      $.util.antiAffinity,

    local service = $.core.v1.service,

    service:
      $.util.serviceFor(self.statefulSet) +
      service.mixin.spec.withClusterIp('None'),
  },

  // Dedicated memcached instance used to cache query results.
  memcached_all: $.memcached {
    name: 'memcached',
    max_item_size: '5m',
  },
}
