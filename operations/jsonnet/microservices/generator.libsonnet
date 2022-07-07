{
  local k = import 'ksonnet-util/kausal.libsonnet',

  local container = k.core.v1.container,
  local containerPort = k.core.v1.containerPort,
  local deployment = k.apps.v1.deployment,
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
    + pvc.mixin.spec.resources.withRequests({ storage: $._config.generator.pvc_size })
    + pvc.mixin.spec.withAccessModes(['ReadWriteOnce'])
    + pvc.mixin.spec.withStorageClassName($._config.generator.pvc_storage_class)
    + pvc.mixin.metadata.withLabels({ app: target_name })
    + pvc.mixin.metadata.withNamespace($._config.namespace),

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
      volumeMount.new(tempo_data_volume, '/var/tempo'),
      volumeMount.new(tempo_overrides_config_volume, '/overrides'),
    ]) +
    $.util.withResources($._config.metrics_generator.resources) +
    $.util.readinessProbe,

  tempo_metrics_generator_deployment:
    deployment.new(
      target_name,
      $._config.metrics_generator.replicas,
      $.tempo_metrics_generator_container,
      {
        app: target_name,
        [$._config.gossip_member_label]: 'true',
      },
    ) +
    deployment.mixin.spec.strategy.rollingUpdate.withMaxSurge(3) +
    deployment.mixin.spec.strategy.rollingUpdate.withMaxUnavailable(1) +
    deployment.mixin.spec.template.metadata.withAnnotations({
      config_hash: std.md5(std.toString($.tempo_metrics_generator_configmap.data['tempo.yaml'])),
    }) +
    deployment.mixin.spec.template.spec.withVolumes([
      volume.fromConfigMap(tempo_config_volume, $.tempo_metrics_generator_configmap.metadata.name),
      volume.fromConfigMap(tempo_overrides_config_volume, $._config.overrides_configmap_name),
      volume.fromPersistentVolumeClaim(tempo_data_volume, $.tempo_metrics_generator_pvc.metadata.name),
    ]),

  tempo_metrics_generator_service:
    k.util.serviceFor($.tempo_metrics_generator_deployment),
}
