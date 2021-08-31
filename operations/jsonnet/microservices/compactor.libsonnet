{
  local k = import 'ksonnet-util/kausal.libsonnet',
  local container = k.core.v1.container,
  local containerPort = k.core.v1.containerPort,
  local volumeMount = k.core.v1.volumeMount,
  local deployment = k.apps.v1.deployment,
  local volume = k.core.v1.volume,
  local service = k.core.v1.service,
  local servicePort = k.core.v1.servicePort,

  local target_name = 'compactor',
  local tempo_config_volume = 'tempo-conf',
  local tempo_data_volume = 'tempo-data',
  local tempo_overrides_config_volume = 'overrides',

  tempo_compactor_container::
    container.new(target_name, $._images.tempo) +
    container.withPorts([
      containerPort.new('prom-metrics', $._config.port),
    ]) +
    container.withArgs([
      '-target=' + target_name,
      '-config.file=/conf/tempo.yaml',
      '-mem-ballast-size-mbs=' + $._config.ballast_size_mbs,
    ]) +
    container.withVolumeMounts([
      volumeMount.new(tempo_config_volume, '/conf'),
      volumeMount.new(tempo_overrides_config_volume, '/overrides'),
    ]) +
    $.util.withResources($._config.compactor.resources) +
    $.util.readinessProbe,

  tempo_compactor_deployment:
    deployment.new(target_name,
                   $._config.compactor.replicas,
                   [
                     $.tempo_compactor_container,
                   ],
                   { app: target_name }) +
    deployment.mixin.spec.strategy.rollingUpdate.withMaxSurge('50%') +
    deployment.mixin.spec.strategy.rollingUpdate.withMaxUnavailable('100%') +
    deployment.mixin.spec.template.metadata.withAnnotations({
      config_hash: std.md5(std.toString($.tempo_compactor_configmap.data['tempo.yaml'])),
    }) +
    deployment.mixin.spec.template.spec.withVolumes([
      volume.fromConfigMap(tempo_config_volume, $.tempo_compactor_configmap.metadata.name),
      volume.fromConfigMap(tempo_overrides_config_volume, $._config.overrides_configmap_name),
    ]),

  tempo_compactor_service:
    k.util.serviceFor($.tempo_compactor_deployment),
}
