{
  local configMap = $.core.v1.configMap,

  tempo_config:: {
    server: {
      http_listen_port: $._config.port
    },
    distributor: {
      receivers: $._config.receivers
    },
    ingester: {
    },
    compactor: {
      compaction: {
        compacted_block_retention: "24h",
      }
    },
    storage: {
      trace: {
        backend: "local",
        wal: {
          path: "/var/tempo/wal",
        },
        'local': {
          path: "/tmp/tempo/traces"
        },
      }
    },
  },

  tempo_configmap:
    configMap.new('tempo') +
    configMap.withData({
      'tempo.yaml': $.util.manifestYaml($.tempo_config),
    }) +
    configMap.withDataMixin({
      'overrides.yaml': |||
        overrides:
      |||,
    }),

  tempo_query_configmap:
    configMap.new('tempo-query') +
    configMap.withData({
      'tempo-query.yaml': $.util.manifestYaml({
        backend: 'localhost:%d' % $._config.port
      })
    }),
}
