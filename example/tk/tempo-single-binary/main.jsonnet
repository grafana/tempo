local tempo = import '../../../operations/jsonnet/single-binary/tempo.libsonnet';
local load = import 'synthetic-load-generator/main.libsonnet';
local grafana = import 'grafana/main.libsonnet';

grafana + load + tempo {
    _images+:: {
        // override images here if desired
    },

    _config +:: {
        namespace: 'default',
        pvc_size: '30Gi',
        pvc_storage_class: 'local-path',
        receivers: {
            jaeger: {
              protocols: {
                thrift_http: null
              }
            }
        }
    },

    local k = import 'ksonnet-util/kausal.libsonnet',
    local container = k.core.v1.container,
    local containerPort = k.core.v1.containerPort,
    tempo_container+::
        container.withPortsMixin([
            containerPort.new('jaeger-http', 14268),
        ]),

    local ingress = k.networking.v1.ingress,
    local rule = k.networking.v1.ingressRule,
    local path = k.networking.v1.httpIngressPath,
    ingress:
        ingress.new('ingress') +
        ingress.mixin.metadata
            .withAnnotationsMixin({
                'ingress.kubernetes.io/ssl-redirect': 'false'
            }) +
        ingress.mixin.spec.withRules(
            rule.http.withPaths([
                path.withPath('/')
                + path.withPathType('ImplementationSpecific')
                + path.backend.service.withName('grafana')
                + path.backend.service.port.withNumber(3000)
            ]),
        ),
}
