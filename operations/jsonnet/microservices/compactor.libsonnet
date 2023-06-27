{
  local target_name = 'compactor',

  local component = import 'component.libsonnet',
  local target = component.newTempoComponent(target_name, image=$._images.tempo, port=$._config.port)
                 + component.withConfigData($.tempo_compactor_config)
                 + component.withDeployment()
                 + component.withGlobalConfig($._config)
                 + component.withReplicas($._config.compactor.replicas)
                 + component.withResources($._config.compactor.resources)
                 + component.withGoMemLimit()
  ,

  compactor:
    target
    {
      // Backwards compatibility for user overrides
      container+: $.tempo_compactor_container,
      deployment+: $.tempo_compactor_deployment,
      service+: $.tempo_compactor_service,
    }
  ,

  tempo_compactor_container:: {},
  tempo_compactor_deployment:: {},
  tempo_compactor_service:: {},
}
