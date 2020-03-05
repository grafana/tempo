{
  local container = $.core.v1.container,
  local containerPort = $.core.v1.containerPort,
  local volumeMount = $.core.v1.volumeMount,
  local pv = $.core.v1.persistentVolume,
  local deployment = $.apps.v1.deployment,
  local volume = $.core.v1.volume,
  local service = $.core.v1.service,
  local servicePort = service.mixin.spec.portsType,

  local target_name = "distributor",
  local frigg_config_volume = 'frigg-conf',

  frigg_distributor_container::
    container.new(target_name, $._images.frigg) +
    container.withPorts([
      containerPort.new('prom-metrics', $._config.port),
    ]) +
    container.withArgs([
      '-target=' + target_name,
      '-config.file=/conf/frigg.yaml',
      '-mem-ballast-size-mbs=' + $._config.ballast_size_mbs,
    ]) +
    container.withVolumeMounts([
      volumeMount.new(frigg_config_volume, '/conf'),
    ]),

  frigg_distributor_deployment:
    deployment.new(target_name,
                   $._config.distributor.replicas,
                   [
                     $.frigg_distributor_container,
                   ],
                   { app: target_name }) +
    deployment.mixin.spec.template.metadata.withAnnotations({
      config_hash: std.md5(std.toString($.frigg_configmap)),
    }) +
    deployment.mixin.spec.template.spec.withVolumes([
      volume.fromConfigMap(frigg_config_volume, $.frigg_configmap.metadata.name),
    ]),

  frigg_distributor_service:
    $.util.serviceFor($.frigg_distributor_deployment)
}
