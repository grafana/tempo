{
  local container = $.core.v1.container,
  local containerPort = $.core.v1.containerPort,
  local volumeMount = $.core.v1.volumeMount,
  local deployment = $.apps.v1.deployment,
  local volume = $.core.v1.volume,
  local service = $.core.v1.service,
  local servicePort = $.core.v1.servicePort,

  local target_name = 'querier',
  local tempo_config_volume = 'tempo-conf',

  tempo_querier_container::
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
    ]) +
    $.util.readinessProbe,

  tempo_querier_deployment:
    deployment.new(
      target_name,
      $._config.querier.replicas,
      $.tempo_querier_container,
      {
        app: target_name,
        [$._config.gossip_member_label]: 'true',
      }
    ) +
    deployment.mixin.spec.strategy.rollingUpdate.withMaxSurge(0) +
    deployment.mixin.spec.strategy.rollingUpdate.withMaxUnavailable(1) +
    deployment.mixin.spec.template.metadata.withAnnotations({
      config_hash: std.md5(std.toString($.tempo_querier_configmap.data['tempo.yaml'])),
    }) +
    deployment.mixin.spec.template.spec.withVolumes([
      volume.fromConfigMap(tempo_config_volume, $.tempo_querier_configmap.metadata.name),
    ]),
}
