{
  local k = import 'k.libsonnet',
  local kausal = import 'ksonnet-util/kausal.libsonnet',

  local service = k.core.v1.service,

  local target_name = 'distributor',
  local this = self,

  local component = import 'component.libsonnet',
  local target = component.newTempoComponent(target_name, image=$._images.tempo, port=$._config.port)
                 + component.withConfigData($.tempo_distributor_config)
                 + component.withDeployment()
                 + component.withGlobalConfig($._config)
                 + component.withGossipSelector()
                 + component.withReplicas($._config.distributor.replicas)
                 + component.withResources($._config.distributor.resources)
                 + component.withSlowRollout()
  ,

  distributor:
    target
    // Backwards compatibility for user overrides
    {
      container+: $.tempo_distributor_container,
      deployment+: $.tempo_distributor_deployment,
      service+: $.tempo_distributor_service,
    },

  tempo_distributor_container:: {},
  tempo_distributor_deployment:: {},
  tempo_distributor_service:: {},

  ingest_service:
    kausal.util.serviceFor(this.distributor.workload)
    + service.mixin.metadata.withName('ingest')
    + service.mixin.spec.withClusterIP('None'),
}
