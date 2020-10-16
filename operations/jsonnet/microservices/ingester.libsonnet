{
  local container = $.core.v1.container,
  local containerPort = $.core.v1.containerPort,
  local volumeMount = $.core.v1.volumeMount,
  local pvc = $.core.v1.persistentVolumeClaim,
  local statefulset = $.apps.v1.statefulSet,
  local volume = $.core.v1.volume,
  local service = $.core.v1.service,
  local servicePort = $.core.v1.servicePort,

  local target_name = 'ingester',
  local tempo_config_volume = 'tempo-conf',
  local tempo_data_volume = 'ingester-data',

  tempo_ingester_pvc::
    pvc.new()
    + pvc.mixin.metadata.withName(tempo_data_volume)
    + pvc.mixin.spec.resources.withRequests({ storage: $._config.ingester.pvc_size })
    + pvc.mixin.spec.withAccessModes(['ReadWriteOnce'])
    + pvc.mixin.spec.withStorageClassName($._config.ingester.pvc_storage_class)
    + pvc.mixin.metadata.withLabels({ app: target_name })
    + pvc.mixin.metadata.withNamespace($._config.namespace),

  tempo_ingester_container::
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
      volumeMount.new(tempo_data_volume, '/var/tempo'),
    ]),

  tempo_ingester_statefulset:
    statefulset.new(
      target_name,
      $._config.ingester.replicas,
      self.tempo_ingester_container,
      self.tempo_ingester_pvc,
      {
        app: target_name,
        [$._config.gossip_member_label]: 'true',
      },
    )
    + statefulset.mixin.spec.withServiceName(target_name)
    + statefulset.mixin.spec.template.spec.withVolumes([
      volume.fromConfigMap(tempo_config_volume, $.tempo_configmap.metadata.name),
    ]),

  tempo_ingester_service:
    $.util.serviceFor($.tempo_ingester_statefulset),

  gossip_ring_service:
    service.new(
      'gossip-ring',  // name
      {
        [$._config.gossip_member_label]: 'true',
      },
      [
        servicePort.newNamed('gossip-ring', $._config.gossip_ring_port, $._config.gossip_ring_port) +
        servicePort.withProtocol('TCP'),
      ],
    ) + service.mixin.spec.withClusterIp('None'),  // headless service
}
