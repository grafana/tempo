local tempo = import '../../../operations/jsonnet/microservices/tempo.libsonnet';
local load = import 'synthetic-load-generator/main.libsonnet';
local minio = import 'minio/minio.libsonnet';

minio + load + tempo {
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
        vulture+:{
            replicas: 0,
        },
        backend: 's3',
        bucket: 'tempo',
    },

    // manually overriding to get tempo to talk to minio
    tempo_config +:: {
        auth_enabled: false,
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
    
    local service = $.core.v1.service,
    tempo_service:
        $.util.serviceFor($.tempo_distributor_deployment)
        + service.mixin.metadata.withName('tempo'),

    local container = $.core.v1.container,
    local containerPort = $.core.v1.containerPort,
    tempo_compactor_container+::
        $.util.resourcesRequests('500m', '500Mi'),

    tempo_distributor_container+::
        $.util.resourcesRequests('500m', '500Mi') +
        container.withPortsMixin([
            containerPort.new('opencensus', 55678),
            containerPort.new('jaeger-http', 14268),
        ]),

    tempo_ingester_container+::
        $.util.resourcesRequests('500m', '500Mi'),

    tempo_querier_container+::
        $.util.resourcesRequests('500m', '500Mi'),

    local ingress = $.extensions.v1beta1.ingress,
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
                ingress.mixin.specType.mixin.backend.withServiceName('querier') +
                ingress.mixin.specType.mixin.backend.withServicePort(16686)
            ),
        ),
}
