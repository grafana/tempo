{
  local container = $.core.v1.container,
  local containerPort = $.core.v1.containerPort,
  local volumeMount = $.core.v1.volumeMount,
  local pvc = $.core.v1.persistentVolumeClaim,
  local statefulset = $.apps.v1.statefulSet,
  local volume = $.core.v1.volume,
  local service = $.core.v1.service,
  local servicePort = service.mixin.spec.portsType,

  local target_name = "ingester",
  local frigg_config_volume = 'frigg-conf',
  local frigg_data_volume = 'ingester-data',

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
    .withName(frigg_data_volume) +
    { kind: 'PersistentVolumeClaim', apiVersion: 'v1' },

  frigg_ingester_container::
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
      volumeMount.new(frigg_data_volume, '/var/frigg'),
    ]),

  frigg_ingester_statefulset:
    statefulset.new(target_name,
                   $._config.ingester.replicas,
                   [
                     $.frigg_ingester_container,
                   ],
                   [
                     $.frigg_ingester_pvc
                   ],
                   { app: target_name })
    .withServiceName(target_name) +
    statefulset.mixin.spec.template.spec.withVolumes([
      volume.fromConfigMap(frigg_config_volume, $.frigg_configmap.metadata.name),
    ]),

  frigg_ingester_service:
    $.util.serviceFor($.frigg_ingester_statefulset),

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
