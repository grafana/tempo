{
  local k = import 'ksonnet-util/kausal.libsonnet',
  local container = k.core.v1.container,
  local containerPort = k.core.v1.containerPort,
  local volumeMount = k.core.v1.volumeMount,
  local pvc = k.core.v1.persistentVolumeClaim,
  local statefulset = k.apps.v1.statefulSet,
  local volume = k.core.v1.volume,
  local service = k.core.v1.service,
  local servicePort = k.core.v1.servicePort,

  local target_name = 'ingester',
  local tempo_config_volume = 'tempo-conf',
  local tempo_data_volume = 'ingester-data',
  local tempo_overrides_config_volume = 'overrides',

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
    (if $._config.variables_expansion then container.withEnvMixin($._config.variables_expansion_env_mixin) else {}) +
    container.withVolumeMounts([
      volumeMount.new(tempo_config_volume, '/conf'),
      volumeMount.new(tempo_data_volume, '/var/tempo'),
      volumeMount.new(tempo_overrides_config_volume, '/overrides'),
    ]) +
    $.util.withResources($._config.ingester.resources) +
    $.util.readinessProbe +
    (if $._config.variables_expansion then container.withArgsMixin(['--config.expand-env=true']) else {}),

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
    + k.util.antiAffinityStatefulSet
    + statefulset.mixin.spec.withServiceName(target_name)
    + statefulset.mixin.spec.template.metadata.withAnnotations({
      config_hash: std.md5(std.toString($.tempo_ingester_configmap.data['tempo.yaml'])),
    })
    + statefulset.mixin.spec.template.spec.withVolumes([
      volume.fromConfigMap(tempo_config_volume, $.tempo_ingester_configmap.metadata.name),
      volume.fromConfigMap(tempo_overrides_config_volume, $._config.overrides_configmap_name),
    ])
    + statefulset.mixin.spec.withPodManagementPolicy('Parallel'),

  tempo_ingester_service:
    k.util.serviceFor($.tempo_ingester_statefulset),

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
