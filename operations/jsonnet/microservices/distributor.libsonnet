{
  local container = $.core.v1.container,
  local containerPort = $.core.v1.containerPort,
  local volumeMount = $.core.v1.volumeMount,
  local pv = $.core.v1.persistentVolume,
  local deployment = $.apps.v1.deployment,
  local volume = $.core.v1.volume,
  local service = $.core.v1.service,
  local servicePort = $.core.v1.servicePort,

  local target_name = 'distributor',
  local tempo_config_volume = 'tempo-conf',

  tempo_distributor_container::
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

  tempo_distributor_deployment:
    deployment.new(target_name,
                   $._config.distributor.replicas,
                   [
                     $.tempo_distributor_container,
                   ],
                   {
                     app: target_name,
                     [$._config.gossip_member_label]: 'true',
                   }) +
    deployment.mixin.spec.template.spec.withTerminationGracePeriodSeconds(60) +
    deployment.mixin.spec.template.metadata.withAnnotations({
      config_hash: std.md5(std.toString($.tempo_configmap.data['tempo.yaml'])),
    }) +
    deployment.mixin.spec.template.spec.withVolumes([
      volume.fromConfigMap(tempo_config_volume, $.tempo_configmap.metadata.name),
    ]),

  tempo_distributor_service:
    $.util.serviceFor($.tempo_distributor_deployment),

  ingest_service:
    $.util.serviceFor($.tempo_distributor_deployment)
    + service.mixin.metadata.withName('ingest')
    + service.mixin.spec.withClusterIp('None'),
}
