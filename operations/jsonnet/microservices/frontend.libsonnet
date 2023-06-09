{
  local k = import 'k.libsonnet',
  local service = k.core.v1.service,
  local servicePort = k.core.v1.servicePort,

  local target_name = 'query-frontend',

  local component = import 'component.libsonnet',
  local target = component.newTempoComponent(target_name)
                 + component.withConfigData($.tempo_query_frontend_config)
                 + component.withDeployment()
                 + component.withDiscoveryService()
                 + component.withGlobalConfig($._config)
                 + component.withReplicas($._config.query_frontend.replicas)
                 + component.withResources($._config.query_frontend.resources)
                 + component.withSlowRollout()
                 + {
                   service+:
                     service.spec.withPortsMixin([
                       servicePort.withName('http')
                       + servicePort.withPort(80)
                       + servicePort.withTargetPort($._config.port),
                     ]),
                 }
  ,

  frontend:
    target
    {
      // Backwards compatibility for user overrides
      container+: $.tempo_query_frontend_container,
      deployment+: $.tempo_query_frontend_deployment,
      service+: $.tempo_query_frontend_service,
      discoveryService+: $.tempo_query_frontend_discovery_service,
    }
  ,

  tempo_query_frontend_container:: {},
  tempo_query_frontend_deployment:: {},
  tempo_query_frontend_service:: {},
  tempo_query_frontend_discovery_service:: {},
}
