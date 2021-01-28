{
  local container = $.core.v1.container,
  local containerPort = $.core.v1.containerPort,
  local volumeMount = $.core.v1.volumeMount,
  local deployment = $.apps.v1.deployment,
  local volume = $.core.v1.volume,
  local service = $.core.v1.service,
  local servicePort = $.core.v1.servicePort,

  local target_name = 'compactor',
  local tempo_config_volume = 'tempo-conf',
  local tempo_data_volume = 'tempo-data',

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
    ]),

  tempo_compactor_deployment:
    deployment.new(target_name,
                   $._config.compactor.replicas,
                   [
                     $.tempo_compactor_container,
                   ],
                   { app: target_name }) +
    deployment.mixin.spec.strategy.rollingUpdate.withMaxSurge(0) +
    deployment.mixin.spec.strategy.rollingUpdate.withMaxUnavailable(1) +
    deployment.mixin.spec.template.metadata.withAnnotations({
      config_hash: std.md5(std.toString($.tempo_compactor_configmap)),
    }) +
    deployment.mixin.spec.template.spec.withVolumes([
      volume.fromConfigMap(tempo_config_volume, $.tempo_compactor_configmap.metadata.name),
    ]),

  tempo_compactor_service:
    $.util.serviceFor($.tempo_compactor_deployment),
}
