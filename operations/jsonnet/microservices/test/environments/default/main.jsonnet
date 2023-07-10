local tempo = import '../../../tempo.libsonnet';

tempo {
  _images+:: {
    // images can be overridden here if desired
  },

  _config+:: {
    cluster: 'k3d',
    namespace: 'default',
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
        opencensus: null,
        jaeger: {
          protocols: {
            thrift_http: null,
          },
        },
      },
    },
    metrics_generator+: {
      pvc_size: '5Gi',
      pvc_storage_class: 'local-path',
      ephemeral_storage_limit_size: '2Gi',
      ephemeral_storage_request_size: '1Gi',
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

    overrides_configmap_name: 'tempo-overrides',
    overrides+:: {},
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
      containerPort.new('opencensus', 55678),
      containerPort.new('jaeger-http', 14268),
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
