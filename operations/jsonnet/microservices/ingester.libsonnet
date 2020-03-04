{
  local container = $.core.v1.container,
  local containerPort = $.core.v1.containerPort,
  local volumeMount = $.core.v1.volumeMount,
  local pvc = $.core.v1.persistentVolumeClaim,
  local deployment = $.apps.v1.deployment,
  local volume = $.core.v1.volume,
  local service = $.core.v1.service,
  local servicePort = service.mixin.spec.portsType,

  local target_name = "ingester",
  local frigg_config_volume = 'frigg-conf',
  local frigg_data_volume = 'frigg-data',

  frigg_ingester_pvc:
    pvc.new() +
    pvc.mixin.spec.resources
    .withRequests({ storage: $._config.ingester.pvc_size }) +
    pvc.mixin.spec
    .withAccessModes(['ReadWriteOnce'])
    .withStorageClassName($._config.ingester.pvc_storage_class) +
    pvc.mixin.metadata
    .withLabels({ app: target_name })
    .withNamespace($._config.namespace)
    .withName(target_name) +
    { kind: 'PersistentVolumeClaim', apiVersion: 'v1' },

  frigg_ingester_container::
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

  frigg_ingester_deployment:
    deployment.new(target_name,
                   1,
                   [
                     $.frigg_ingester_container,
                   ],
                   { app: target_name }) +
    deployment.mixin.spec.template.metadata.withAnnotations({
      config_hash: std.md5(std.toString($.frigg_configmap)),
    }) +
    deployment.mixin.spec.template.spec.withVolumes([
      volume.fromConfigMap(frigg_config_volume, $.frigg_configmap.metadata.name),
      volume.fromPersistentVolumeClaim(frigg_data_volume, $.frigg_compactor_pvc.metadata.name),
    ]),

  frigg_service:
    $.util.serviceFor($.frigg_ingester_deployment),

  gossip_ring_service:
    service.new(
      'gossip-ring',  // name
      { app: target_name },
      [
        servicePort.newNamed('gossip-ring', $._config.gossip_ring_port, $._config.gossip_ring_port) +
        servicePort.withProtocol('TCP'),
      ],
      ) + service.mixin.spec.withClusterIp('None'),  // headless service
}
