local tempo = import '../../../operations/jsonnet/microservices/tempo.libsonnet';
local load = import 'synthetic-load-generator/main.libsonnet';
local grafana = import 'grafana/main.libsonnet';
local minio = import 'minio/minio.libsonnet';

minio + grafana + load + tempo {
    _images+:: {
        // images can be overridden here if desired
    },

    _config +:: {
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
        memcached+: {
          replicas: 1,
        },
        vulture+: {
            replicas: 0,
        },
        backend: 's3',
        bucket: 'tempo',
        tempo_query_url: 'http://query-frontend:3200',
    },

    // manually overriding to get tempo to talk to minio
    tempo_config +:: {
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
                    affinity: {}
                }
            }
        }
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
                        affinity: {}
                    }
                }
            }
        }
    },

    local ingress = k.networking.v1beta1.ingress,
    ingress:
        ingress.new() +
        ingress.mixin.metadata
            .withName('ingress')
            .withAnnotations({
                'ingress.kubernetes.io/ssl-redirect': 'false'
            }) +
        ingress.mixin.spec.withRules(
            ingress.mixin.specType.rulesType.mixin.http.withPaths(
                ingress.mixin.spec.rulesType.mixin.httpType.pathsType.withPath('/') +
                ingress.mixin.specType.mixin.backend.withServiceName('grafana') +
                ingress.mixin.specType.mixin.backend.withServicePort(3000)
            ),
        ),
}
