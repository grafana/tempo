{
  local k = import 'ksonnet-util/kausal.libsonnet',
  local container = k.core.v1.container,
  local containerPort = k.core.v1.containerPort,
  local volumeMount = k.core.v1.volumeMount,
  local deployment = k.apps.v1.deployment,
  local volume = k.core.v1.volume,
  local service = k.core.v1.service,
  local servicePort = k.core.v1.servicePort,

  local target_name = 'query-frontend',
  local tempo_config_volume = 'tempo-conf',
  local tempo_query_config_volume = 'tempo-query-conf',
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
    $.util.withResources($._config.query_frontend.resources) +
    $.util.readinessProbe,

  tempo_query_container::
    container.new('tempo-query', $._images.tempo_query) +
    container.withPorts([
      containerPort.new('jaeger-ui', 16686),
      containerPort.new('jaeger-metrics', 16687),
    ]) +
    container.withArgs([
      '--query.base-path=' + $._config.jaeger_ui.base_path,
      '--grpc-storage-plugin.configuration-file=/conf/tempo-query.yaml',
      '--query.bearer-token-propagation=true',
    ]) +
    container.withVolumeMounts([
      volumeMount.new(tempo_query_config_volume, '/conf'),
    ]),

  tempo_query_frontend_deployment:
    deployment.new(
      target_name,
      $._config.query_frontend.replicas,
      [$.tempo_query_frontend_container, $.tempo_query_container],
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
      volume.fromConfigMap(tempo_query_config_volume, $.tempo_query_configmap.metadata.name),
      volume.fromConfigMap(tempo_config_volume, $.tempo_query_frontend_configmap.metadata.name),
    ]),

  tempo_query_frontend_service:
    k.util.serviceFor($.tempo_query_frontend_deployment)
    + service.mixin.spec.withPortsMixin([
      servicePort.withName('http')
      + servicePort.withPort(80)
      + servicePort.withTargetPort($._config.port),
    ]),

  tempo_query_frontend_discovery_service:
    k.util.serviceFor($.tempo_query_frontend_deployment)
    + service.mixin.spec.withPortsMixin([
      servicePort.withName('grpc')
      + servicePort.withPort(9095)
      + servicePort.withTargetPort(9095),
    ])
    + service.mixin.spec.withPublishNotReadyAddresses(true)
    + service.mixin.spec.withClusterIp('None')
    + service.mixin.metadata.withName('query-frontend-discovery'),
}
