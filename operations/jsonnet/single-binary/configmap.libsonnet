{
  local k = import 'ksonnet-util/kausal.libsonnet',
  local configMap = k.core.v1.configMap,

  tempo_config:: {
    server: {
      http_listen_port: $._config.tempo.port,
    },
    distributor: {
      receivers: $._config.receivers,
    },
    ingester: {
    },
    compactor: {
      compaction: {
        block_retention: '24h',
      },
    },
    memberlist: {
      abort_if_cluster_join_fails: false,
      bind_port: 7946,
      join_members: [
        '%s:7946' % $._config.tempo.headless_service_name,
      ],
    },
    storage: {
      trace: {
        backend: 'local',
        wal: {
          path: '/var/tempo/wal',
        },
        'local': {
          path: '/var/tempo/traces',
        },
      },
    },
    querier: {
      frontend_worker: {
        frontend_address: 'tempo:9095',
      },
    },
  },

  tempo_configmap:
    configMap.new('tempo') +
    configMap.withData({
      'tempo.yaml': k.util.manifestYaml($.tempo_config),
    }) +
    configMap.withDataMixin({
      'overrides.yaml': |||
        overrides:
      |||,
    }),

  tempo_query_configmap: if $._config.tempo_query.enabled then
    configMap.new('tempo-query') +
    configMap.withData({
      'tempo-query.yaml': k.util.manifestYaml({
        backend: 'localhost:%d' % $._config.tempo.port,
      }),
    }),
}
