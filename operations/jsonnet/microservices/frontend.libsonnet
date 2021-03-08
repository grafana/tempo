{
  local container = $.core.v1.container,
  local containerPort = $.core.v1.containerPort,
  local volumeMount = $.core.v1.volumeMount,
  local deployment = $.apps.v1.deployment,
  local volume = $.core.v1.volume,
  local service = $.core.v1.service,
  local servicePort = $.core.v1.servicePort,

  local target_name = 'query-frontend',
  local tempo_config_volume = 'tempo-conf',
  local tempo_data_volume = 'tempo-data',

  tempo_query_frontend_container::
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

  tempo_query_frontend_deployment:
    deployment.new(
      target_name,
      $._config.query_frontend.replicas,
      $.tempo_query_frontend_container,
      {
        app: target_name,
      }
    ) +
    deployment.mixin.spec.strategy.rollingUpdate.withMaxSurge(0) +
    deployment.mixin.spec.strategy.rollingUpdate.withMaxUnavailable(1) +
    deployment.mixin.spec.template.metadata.withAnnotations({
      config_hash: std.md5(std.toString($.tempo_query_frontend_configmap.data['tempo.yaml'])),
    }) +
    deployment.mixin.spec.template.spec.withVolumes([
      volume.fromConfigMap(tempo_config_volume, $.tempo_query_frontend_configmap.metadata.name),
    ]),

  tempo_query_frontend_service:
    $.util.serviceFor($.tempo_query_frontend_deployment)
    + service.mixin.spec.withPortsMixin([
      servicePort.withName('http')
      + servicePort.withPort(80)
      + servicePort.withTargetPort(3100),
    ]),

  tempo_query_frontend_discovery_service:
    $.util.serviceFor($.tempo_query_frontend_deployment)
    + service.mixin.spec.withPortsMixin([
      servicePort.withName('grpc')
      + servicePort.withPort(9095)
      + servicePort.withTargetPort(9095),
    ])
    + service.mixin.spec.withPublishNotReadyAddresses(true)
    + service.mixin.spec.withClusterIp('None')
    + service.mixin.metadata.withName('query-frontend-discovery'),
}
