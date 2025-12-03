local memcached = import 'memcached/memcached.libsonnet';

memcached {
  local this = self,

  _config+:: {
    memcached_tiered_enabled: false,
  },

  tempo_config+:: if !$._config.memcached_tiered_enabled then
    {
      storage+: {
        trace+: {
          cache: 'memcached',
          memcached: {
            consistent_hash: true,
            timeout: '200ms',
            host: 'memcached',
            service: 'memcached-client',
          },
        },
      },
    },

  memcached+:: {
    cpu_limits:: null,
    cpu_requests:: '100m',
    connection_limit: $._config.memcached.connection_limit,
    memory_limit_mb: $._config.memcached.memory_limit_mb,
    replicas:: 0,
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

  memcached_frontend_search:
    if this._config.memcached_tiered_enabled then
      $.memcached {
        name: 'memcached-frontend-search',

        replicas: $._config.memcached_frontend_search.replicas,
        connection_limit: $._config.memcached_frontend_search.connection_limit,
        max_item_size: $._config.memcached_frontend_search.cache_max_size_mbs + 'm',
        memory_limit_mb: $._config.memcached_frontend_search.memory_limit_mb,
      }
    else null,

  memcached_parquet_page: if this._config.memcached_tiered_enabled then
    $.memcached {
      name: 'memcached-parquet-page',
      replicas: $._config.memcached_parquet_page.replicas,
      connection_limit: $._config.memcached_parquet_page.connection_limit,
      max_item_size: $._config.memcached_parquet_page.cache_max_size_mbs + 'm',
      memory_limit_mb: $._config.memcached_parquet_page.memory_limit_mb,
    }
  else null,

  memcached_all: $.memcached {
    name: 'memcached',
    replicas: $._config.memcached.replicas,
    connection_limit: $._config.memcached.connection_limit,
    max_item_size: $._config.memcached.cache_max_size_mbs + 'm',
    memory_limit_mb: $._config.memcached.memory_limit_mb,
  },

  caches_config:: [
    // NOTE: bloom and parquet-footer use the same memcached cluster but
    // different clients. different clients force tempo to metric the caches
    // independently for better vision on how each is performing
    {
      roles: ['bloom'],
      memcached: {
        consistent_hash: true,
        timeout: '200ms',
        host: this.memcached_all.name,  // this memcached cluster is defined in the vendored memcached.libsonnet in the OSS tempo repo
        service: 'memcached-client',
        max_idle_conns: 100,
        max_item_size: $._config.memcached.cache_max_size_mbs * 1024 * 1024,
      },
    },
    {
      roles: ['trace-id-index'],
      memcached: {
        consistent_hash: true,
        timeout: '200ms',
        host: this.memcached_all.name,  // this memcached cluster is defined in the vendored memcached.libsonnet in the OSS tempo repo
        service: 'memcached-client',
        max_idle_conns: 100,
        max_item_size: $._config.memcached.cache_max_size_mbs * 1024 * 1024,
      },
    },
    {
      roles: ['parquet-footer'],
      memcached: {
        consistent_hash: true,
        timeout: '200ms',
        host: this.memcached_all.name,  // this memcached cluster is defined in the vendored memcached.libsonnet in the OSS tempo repo
        service: 'memcached-client',
        max_idle_conns: 100,
        max_item_size: $._config.memcached.cache_max_size_mbs * 1024 * 1024,
      },
    },
    {
      roles: ['frontend-search'],
      memcached: {
        consistent_hash: true,
        timeout: '50ms',
        host: this.memcached_frontend_search.name,
        service: 'memcached-client',  // service is the port name in k8s
        max_idle_conns: 100,
        max_item_size: $._config.memcached_frontend_search.cache_max_size_mbs * 1024 * 1024,
      },
    },
    {
      roles: ['parquet-page'],
      memcached: {
        consistent_hash: true,
        timeout: '200ms',
        host: this.memcached_parquet_page.name,
        service: 'memcached-client',  // service is the port name in k8s
        max_idle_conns: 100,
        max_item_size: $._config.memcached_parquet_page.cache_max_size_mbs * 1024 * 1024,
      },
    },
  ],

  // disables cache control on the querier. remove once this is rolled out everywhere and the corresponding setting is removed
  // from tempo/tempo.libsonnet
  tempo_querier_config+:: if $._config.memcached_tiered_enabled then
    {
      cache+: {
        caches+: this.caches_config,
      },
      storage+: {
        trace+: {
          search+: {
            cache_control: {
              footer: false,
            },
          },
        },
      },
    }
  else {},

  tempo_ingester_config+:: if $._config.memcached_tiered_enabled then
    {
      cache+: {
        caches+: this.caches_config,
      },
    }
  else {},

  tempo_query_frontend_config+:: if $._config.memcached_tiered_enabled then
    {
      cache+: {
        caches+: this.caches_config,
      },
    }
  else {},

  tempo_compactor_config+:: if $._config.memcached_tiered_enabled then
    {
      cache+: {
        caches+: this.caches_config,
      },
    }
  else {},

}
