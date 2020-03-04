local frigg = import '../../../operations/jsonnet/microservices/frigg.libsonnet';
local load = import 'synthetic-load-generator/main.libsonnet';

load + frigg {
    _config +:: {
        namespace: 'default',
        compactor+: {
            pvc_size: '5Gi',
            pvc_storage_class: 'local-path',
        },
        querier+: {
            pvc_size: '5Gi',
            pvc_storage_class: 'local-path',
        },
        ingester+: {
            pvc_size: '5Gi',
            pvc_storage_class: 'local-path',
        },
        distributor+: {
            receivers: {
                opencensus: null
            }
        },
    },

    local service = $.core.v1.service,
    frigg_service:
        $.util.serviceFor($.frigg_distributor_deployment)
        + service.mixin.metadata.withName('frigg'),

    local container = $.core.v1.container,
    local containerPort = $.core.v1.containerPort,
    frigg_compactor_container+::
        $.util.resourcesRequests('500m', '500Mi'),

    frigg_distributor_container+::
        $.util.resourcesRequests('500m', '500Mi') +
        container.withPortsMixin([
            containerPort.new('opencensus', 55678),
        ]),

    frigg_ingester_container+::
        $.util.resourcesRequests('500m', '500Mi'),

    frigg_querier_container+::
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
