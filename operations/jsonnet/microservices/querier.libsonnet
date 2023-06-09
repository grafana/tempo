{
  local k = import 'ksonnet-util/kausal.libsonnet',
  local container = k.core.v1.container,
  local containerPort = k.core.v1.containerPort,
  local volumeMount = k.core.v1.volumeMount,
  local deployment = k.apps.v1.deployment,
  local volume = k.core.v1.volume,

  local target_name = 'querier',

  local component = import 'component.libsonnet',
  local target = component.newTempoComponent(target_name)
                 + component.withConfigData($.tempo_querier_config)
                 + component.withDeployment()
                 + component.withGlobalConfig($._config)
                 + component.withGossipSelector()
                 + component.withReplicas($._config.querier.replicas)
                 + component.withResources($._config.querier.resources)
                 + component.withSlowRollout()
  ,

  querier:
    target
    {
      // Backwards compatibility for user overrides
      container+: $.tempo_querier_container,
      deployment+: $.tempo_querier_deployment,
      service+: $.tempo_querier_service,
    },

  tempo_querier_container:: {},
  tempo_querier_deployment:: {},
  tempo_querier_service:: {},
}
