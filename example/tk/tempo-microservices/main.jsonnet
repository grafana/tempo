local tempo = import '../../../operations/jsonnet/microservices/tempo.libsonnet';
local dashboards = import 'dashboards/grafana.libsonnet';
local metrics = import 'metrics/prometheus.libsonnet';
local minio = import 'minio/minio.libsonnet';
local load = import 'synthetic-load-generator/main.libsonnet';

minio + metrics + load + tempo {

  dashboards:
    dashboards.deploy(),

  _images+:: {
    // images can be overridden here if desired
  },

  _config+:: {
    cluster: 'k3d',
    namespace: 'default',
    block_builder_concurrent_rollout_enabled: true,
    compactor+: {
    },
    querier+: {
    },
    ingester+: {
      pvc_size: '5Gi',
      pvc_storage_class: 'local-path',
    },
    distributor+: {
      receivers: {
        otlp: {
          protocols: {
            grpc: null,
          },
        },
      },
    },
    metrics_generator+: {
      replicas: 1,
      ephemeral_storage_limit_size: '2Gi',
      ephemeral_storage_request_size: '1Gi',
      pvc_size: '1Gi',
      pvc_storage_class: 'local-path',
    },
    block_builder+:{
      replicas: 2,
    },
    memcached+: {
      replicas: 1,
    },
    vulture+: {
      replicas: 0,
      // Disable search until release
      tempoSearchBackoffDuration: '0s',
    },
    backend: 's3',
    bucket: 'tempo',
    tempo_query_url: 'http://query-frontend:3200',
  },

  // manually overriding to get tempo to talk to minio
  tempo_config+:: {
    storage+: {
      trace+: {
        s3+: {
          endpoint: 'minio:9000',
          access_key: 'tempo',
          secret_key: 'supersecret',
          insecure: true,
        },
      },
    },
  },

  local k = import 'ksonnet-util/kausal.libsonnet',
  local service = k.core.v1.service,
  tempo_service:
    k.util.serviceFor($.tempo_distributor_deployment)
    + service.mixin.metadata.withName('tempo'),

  local container = k.core.v1.container,
  local containerPort = k.core.v1.containerPort,
  tempo_compactor_container+::
    k.util.resourcesRequests('500m', '500Mi'),

  tempo_distributor_container+::
    k.util.resourcesRequests('500m', '500Mi') +
    container.withPortsMixin([
      containerPort.new('otlp-grpc', 4317),
    ]),

  tempo_ingester_container+::
    k.util.resourcesRequests('500m', '500Mi'),

  // clear affinity so we can run multiple ingesters on a single node
  tempo_ingester_statefulset+: {
    spec+: {
      template+: {
        spec+: {
          affinity: {},
        },
      },
    },
  },

  tempo_querier_container+::
    k.util.resourcesRequests('500m', '500Mi'),

  tempo_query_frontend_container+::
    k.util.resourcesRequests('300m', '500Mi'),

  // clear affinity so we can run multiple instances of memcached on a single node
  memcached_all+: {
    statefulSet+: {
      spec+: {
        template+: {
          spec+: {
            affinity: {},
          },
        },
      },
    },
  },

  local ingress = k.networking.v1.ingress,
  local rule = k.networking.v1.ingressRule,
  local path = k.networking.v1.httpIngressPath,
  ingress:
    ingress.new('ingress') +
    ingress.mixin.metadata
    .withAnnotationsMixin({
      'ingress.kubernetes.io/ssl-redirect': 'false',
    }) +
    ingress.mixin.spec.withRules(
      rule.http.withPaths([
        path.withPath('/')
        + path.withPathType('ImplementationSpecific')
        + path.backend.service.withName('grafana')
        + path.backend.service.port.withNumber(3000),
      ]),
    ),
}
