{
  local container = $.core.v1.container,
  local containerPort = $.core.v1.containerPort,
  local volumeMount = $.core.v1.volumeMount,
  local pvc = $.core.v1.persistentVolumeClaim,
  local deployment = $.apps.v1.deployment,
  local volume = $.core.v1.volume,
  local service = $.core.v1.service,
  local servicePort = service.mixin.spec.portsType,

  local target_name = "querier",
  local frigg_config_volume = 'frigg-conf',
  local frigg_query_config_volume = 'frigg-query-conf',
  local frigg_data_volume = 'frigg-data',

  frigg_querier_pvc:
    pvc.new() +
    pvc.mixin.spec.resources
    .withRequests({ storage: $._config.querier.pvc_size }) +
    pvc.mixin.spec
    .withAccessModes(['ReadWriteOnce'])
    .withStorageClassName($._config.querier.pvc_storage_class) +
    pvc.mixin.metadata
    .withLabels({ app: target_name })
    .withNamespace($._config.namespace)
    .withName(target_name) +
    { kind: 'PersistentVolumeClaim', apiVersion: 'v1' },

  frigg_querier_container::
    container.new(target_name, $._images.frigg) +
    container.withPorts([
      containerPort.new('prom-metrics', 3100),
    ]) +
    container.withArgs([
      '-target=' + target_name,
      '-config.file=/conf/frigg.yaml',
      '-mem-ballast-size-mbs=' + $._config.ballast_size_mbs,
    ]) +
    container.withVolumeMounts([
      volumeMount.new(frigg_config_volume, '/conf'),
      volumeMount.new(frigg_data_volume, '/var/frigg'),
    ]),

  frigg_query_container::
    container.new('frigg-query', $._images.frigg_query) +
    container.withPorts([
      containerPort.new('jaeger-ui', 16686),
    ]) +
    container.withArgs([
      '--query.base-path=' + $._config.jaeger_ui.base_path,
      '--grpc-storage-plugin.configuration-file=/conf/frigg-query.yaml',
    ]) +
    container.withVolumeMounts([
      volumeMount.new(frigg_query_config_volume, '/conf'),
    ]),

  frigg_querier_deployment:
    deployment.new(target_name,
                   $._config.querier.replicas,
                   [
                     $.frigg_querier_container,
                     $.frigg_query_container,
                   ],
                   { app: target_name }) +
    deployment.mixin.spec.template.metadata.withAnnotations({
      config_hash: std.md5(std.toString($.frigg_compactor_configmap)),
    }) +
    deployment.mixin.spec.template.spec.withVolumes([
      volume.fromConfigMap(frigg_query_config_volume, $.frigg_query_configmap.metadata.name),
      volume.fromConfigMap(frigg_config_volume, $.frigg_configmap.metadata.name),
      volume.fromPersistentVolumeClaim(frigg_data_volume, $.frigg_querier_pvc.metadata.name),
    ]),

  frigg_service:
    $.util.serviceFor($.frigg_querier_deployment)
    + service.mixin.spec.withPortsMixin([
      servicePort.withName('http').withPort(80).withTargetPort(16686),
    ]),
}
