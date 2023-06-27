{
  local k = import 'k.libsonnet',
  local kausal = import 'ksonnet-util/kausal.libsonnet',

  local container = k.core.v1.container,
  local containerPort = kausal.core.v1.containerPort,
  local deployment = k.apps.v1.deployment,
  local statefulset = k.apps.v1.statefulSet,
  local volume = k.core.v1.volume,
  local pvc = k.core.v1.persistentVolumeClaim,
  local volumeMount = k.core.v1.volumeMount,

  local target_name = 'metrics-generator',

  local component = import 'component.libsonnet',
  local target = component.newTempoComponent(target_name, image=$._images.tempo, port=$._config.port)
                 + component.withConfigData($.tempo_metrics_generator_config)
                 + component.withStatefulset()
                 + component.withGlobalConfig($._config)
                 + component.withGossipSelector()
                 + component.withReplicas($._config.metrics_generator.replicas)
                 + component.withResources($._config.metrics_generator.resources)
                 + component.withSlowRollout()
                 + component.withEphemeralStorage(
                   $._config.metrics_generator.ephemeral_storage_request_size,
                   $._config.metrics_generator.ephemeral_storage_limit_size,
                   $.tempo_metrics_generator_config.metrics_generator.storage.path,
                 )
  ,

  metrics_generator:
    target
    {
      // Backwards compatibility for user overrides
      container+: $.tempo_metrics_generator_container,
      deployment+: $.tempo_metrics_generator_deployment,
      statefulset+: $.tempo_metrics_generator_statefulset,
      service+: $.tempo_metrics_generator_service,
    }
  ,

  tempo_metrics_generator_container:: {},
  tempo_metrics_generator_deployment:: {},
  tempo_metrics_generator_statefulset:: {},
  tempo_metrics_generator_service:: {},
}
