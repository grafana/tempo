{
  local k = import 'ksonnet-util/kausal.libsonnet',
  local container = k.core.v1.container,
  local containerPort = k.core.v1.containerPort,
  local volumeMount = k.core.v1.volumeMount,
  local deployment = k.apps.v1.deployment,
  local volume = k.core.v1.volume,
  local service = k.core.v1.service,
  local statefulset = k.apps.v1.statefulSet,

  local component = import 'component.libsonnet',

  local target_name = 'distributor',
  local tempo_config_volume = 'tempo-conf',
  local tempo_overrides_config_volume = 'overrides',

  distributor:
    component.new(target_name)
    + component.withDeployment()
    + component.withReplicas($._config.distributor.replicas)
  ,

  tempo_distributor_service:
    k.util.serviceFor($.tempo_distributor_deployment),

  ingest_service:
    k.util.serviceFor($.tempo_distributor_deployment)
    + service.mixin.metadata.withName('ingest')
    + service.mixin.spec.withClusterIp('None'),
}
