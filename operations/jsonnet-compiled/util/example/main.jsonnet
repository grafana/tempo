// The jsonnet file used to generate the Kubernetes manifests.

local tempo = import 'microservices/tempo.libsonnet';

tempo {
  _images+:: {
    tempo: 'grafana/tempo:latest',
    tempo_query: 'grafana/tempo-query:latest',
    tempo_vulture: 'grafana/tempo-vulture:latest',
  },

  _config+:: {
    namespace: 'tracing',
    compactor+: {
      replicas: 5,
    },
    query_frontend+: {
      replicas: 2,
    },
    querier+: {
      replicas: 5,
    },
    ingester+: {
      replicas: 10,
      pvc_size: '10Gi',
      pvc_storage_class: 'fast',
    },
    distributor+: {
      replicas: 5,
      receivers: {
        jaeger: {
          protocols: {
            grpc: {
              endpoint: '0.0.0.0:14250',
            },
          },
        },
        otlp: {
          protocols: {
            grpc: {
              endpoint: '0.0.0.0:4317',
            },
          },
        },
      },
    },
    metrics_generator+: {
      ephemeral_storage_request_size: '10Gi',
      ephemeral_storage_limit_size: '11Gi',
    },
    memcached+: {
      replicas: 5,
    },
    vulture+: {
      replicas: 1,
      tempoOrgId: '1',
      tempoPushUrl: 'http://distributor',
      tempoQueryUrl: 'http://query-frontend:3200/tempo',
    },
    jaeger_ui: {
      base_path: '/tempo',
    },
    backend: 'gcs',
    bucket: 'tempo',
  },

  local statefulSet = $.apps.v1.statefulSet,
  tempo_ingester_statefulset:
    if !$._config.multi_zone_ingester_enabled then super.tempo_ingester_statefulset + statefulSet.mixin.spec.withPodManagementPolicy('Parallel') else null,

}
