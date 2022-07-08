{
  local k = import 'ksonnet-util/kausal.libsonnet',

  local container = k.core.v1.container,
  local containerPort = k.core.v1.containerPort,
  local statefulset = k.apps.v1.statefulSet,
  local pvc = k.core.v1.persistentVolumeClaim,
  local volume = k.core.v1.volume,
  local volumeMount = k.core.v1.volumeMount,

  local target_name = 'metrics-generator',
  local tempo_config_volume = 'tempo-conf',
  local tempo_data_volume = 'metrics-generator-data',
  local tempo_overrides_config_volume = 'overrides',

  tempo_metrics_generator_pvc::
    pvc.new()
    + pvc.mixin.metadata.withName(tempo_data_volume)
    + pvc.mixin.metadata.withLabels({ app: target_name })
    + pvc.mixin.metadata.withNamespace($._config.namespace)
    // waiting for k 1.23
    // + pvc.mixin.spec.persistentVolumeClaimRetentionPolicy.withWhenScaled('delete')
    + pvc.mixin.spec.resources.withRequests({ storage: $._config.metrics_generator.pvc_size })
    + pvc.mixin.spec.withAccessModes(['ReadWriteOnce'])
    + pvc.mixin.spec.withStorageClassName($._config.metrics_generator.pvc_storage_class),

  tempo_metrics_generator_container::
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
      volumeMount.new(tempo_data_volume, $.tempo_metrics_generator_config.metrics_generator.storage.path),
      volumeMount.new(tempo_overrides_config_volume, '/overrides'),
    ]) +
    $.util.withResources($._config.metrics_generator.resources) +
    $.util.readinessProbe,

  tempo_metrics_generator_statefulset:
    statefulset.new(
      target_name,
      $._config.metrics_generator.replicas,
      $.tempo_metrics_generator_container,
      $.tempo_metrics_generator_pvc,
      {
        app: target_name,
        [$._config.gossip_member_label]: 'true',
      },
    ) +
    statefulset.mixin.spec.template.metadata.withAnnotations({
      config_hash: std.md5(std.toString($.tempo_metrics_generator_configmap.data['tempo.yaml'])),
    }) +
    statefulset.mixin.spec.template.spec.withVolumes([
      volume.fromConfigMap(tempo_config_volume, $.tempo_metrics_generator_configmap.metadata.name),
      volume.fromConfigMap(tempo_overrides_config_volume, $._config.overrides_configmap_name),
    ]),

  tempo_metrics_generator_service:
    k.util.serviceFor($.tempo_metrics_generator_statefulset),
}
