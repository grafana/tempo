{
  local container = $.core.v1.container,
  local containerPort = $.core.v1.containerPort,
  local volumeMount = $.core.v1.volumeMount,
  local pvc = $.core.v1.persistentVolumeClaim,
  local deployment = $.apps.v1.deployment,
  local volume = $.core.v1.volume,
  local service = $.core.v1.service,
  local servicePort = service.mixin.spec.portsType,

  local target_name = "compactor",
  local frigg_config_volume = 'frigg-conf',
  local frigg_data_volume = 'frigg-data',

  frigg_compactor_pvc:
    pvc.new() +
    pvc.mixin.spec.resources
    .withRequests({ storage: $._config.compactor.pvc_size }) +
    pvc.mixin.spec
    .withAccessModes(['ReadWriteOnce'])
    .withStorageClassName($._config.compactor.pvc_storage_class) +
    pvc.mixin.metadata
    .withLabels({ app: target_name })
    .withNamespace($._config.namespace)
    .withName(target_name) +
    { kind: 'PersistentVolumeClaim', apiVersion: 'v1' },

  frigg_compactor_container::
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

  frigg_compactor_deployment:
    deployment.new(target_name,
                   $._config.compactor.replicas,
                   [
                     $.frigg_compactor_container,
                   ],
                   { app: target_name }) +
    deployment.mixin.spec.template.metadata.withAnnotations({
      config_hash: std.md5(std.toString($.frigg_compactor_configmap)),
    }) +
    deployment.mixin.spec.template.spec.withVolumes([
      volume.fromConfigMap(frigg_config_volume, $.frigg_compactor_configmap.metadata.name),
      volume.fromPersistentVolumeClaim(frigg_data_volume, $.frigg_compactor_pvc.metadata.name),
    ]),
}
