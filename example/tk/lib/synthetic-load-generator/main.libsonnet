{
  local k = import 'ksonnet-util/kausal.libsonnet',

  local deployment = k.apps.v1.deployment,
  local container = k.core.v1.container,
  local configMap = k.core.v1.configMap,
  local envVar = k.core.v1.envVar,
  local volume = k.core.v1.volume,
  local volumeMount = k.core.v1.volumeMount,

  k6_tracing_data_deployment:
    deployment.new('k6-tracing', 1,
                   [
                     $.k6_tracing_container,
                   ],
                   {
                     app: 'k6-tracing',
                   }) +
    deployment.mixin.spec.template.metadata.withAnnotations({
      config_hash: std.md5(std.toString($.k6_tracing_config_map.data['template.js'])),
    }) +
    deployment.mixin.spec.template.spec.withVolumes([
      volume.fromConfigMap('k6-tracing', 'k6-tracing'),
    ]),

  k6_tracing_container::
    container.new('k6-tracing', 'ghcr.io/grafana/xk6-client-tracing:v0.0.5') +
    container.withArgs([
      'run',
      '/var/k6/template.js',
    ]) +
    container.withEnvMixin([envVar.new('ENDPOINT', 'http://distributor:4318')]) +
    container.withVolumeMountsMixin([
      volumeMount.withName('k6-tracing')
      + volumeMount.withMountPath('/var/k6'),
    ]),


  tempo_config+:: {
    storage+: {
      trace+: {
        empty_tenant_deletion_age: '24h',
        empty_tenant_deletion_enabled: true,
      },
    },
  },

  k6_tracing_config_map:
    configMap.new('k6-tracing') +
    configMap.withData({
      'template.js': |||
        import {sleep} from 'k6';
        import tracing from 'k6/x/tracing';
        import { randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

        export const options = {
            vus: 1,
            duration: "30m",
        };

        const endpoint = __ENV.ENDPOINT || "otel-collector:4317"
        const client = new tracing.Client({
            endpoint,
            exporter: tracing.EXPORTER_OTLP_HTTP,
            tls: {
              insecure: true,
            },
            headers: {
              'X-Scope-OrgID': '3',
            },
        });

        const traceDefaults = {
            attributeSemantics: tracing.SEMANTICS_HTTP,
            randomAttributes: {count: 2, cardinality: 5},
            randomEvents: {count: 0.1, exceptionCount: 0.2, randomAttributes: {count: 6, cardinality: 20}},
        }

        const traceTemplates = [
            {
                defaults: traceDefaults,
                spans: [
                    {service: "gateway", name: "/foo", duration: {min: 200, max: 900}, attributes: {"http.status_code": 200, "application.version": "1.1"}},
                    {service: "gateway", name: "authenticate", duration: {min: 50, max: 100}},
                    {service: "auth-service", name: "/auth", attributes: {"http.status_code": 200, "application.version": "3.1"}},
                    {service: "cache", name: "GET", attributes: {"http.status_code": 200, "application.version": "2.1"}},
                    {service: "identity-service", name: "/user", parentIdx: 2, attributes: {"http.status_code": 200, "application.version": "0.3"}},
                    {service: "service-A", name: "/id", attributes: {"http.status_code": 200, "application.version": "12.3"}},
                    {service: "service-B", name: "/id", parentIdx: 4, attributes: {"http.status_code": 200, "application.version": "9.1"}},
                ]
            },
            {
                defaults: traceDefaults,
                spans: [
                    {service: "gateway", name: "/foo", duration: {min: 200, max: 900}, attributes: {"http.status_code": 500}},
                    {service: "gateway", name: "authenticate", duration: {min: 50, max: 100}},
                    {service: "auth-service", name: "/auth", attributes: {"http.status_code": 500}},
                    {service: "cache", name: "GET", attributes: {"http.status_code": 200, "application.version": "2.1"}},
                    {service: "identity-service", name: "/user", parentIdx: 2, attributes: {"http.status_code": 400, "application.version": "0.4"}},
                    {service: "service-A", name: "/id", attributes: {"http.status_code": 200}},
                    {service: "service-B", name: "/id", parentIdx: 4, attributes: {"http.status_code": 200}},
                ]
            },
        ]

        export default function () {
            const d = new Date();
            let minutes = d.getMinutes();
            let idx = Math.floor(minutes / 5) % traceTemplates.length

            const gen = new tracing.TemplatedGenerator(traceTemplates[idx])
            client.push(gen.traces())

            sleep(randomIntBetween(1, 5));
        }

        export function teardown() {
            client.shutdown();
        }
      |||,
    }),
}
