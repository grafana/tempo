{
  local k = import 'k.libsonnet',
  local kausal = import 'ksonnet-util/kausal.libsonnet',

  local container = k.core.v1.container,
  local containerPort = kausal.core.v1.containerPort,
  local deployment = k.apps.v1.deployment,
  local statefulset = k.apps.v1.statefulSet,
  local volume = k.core.v1.volume,
  local pvc = k.core.v1.persistentVolumeClaim,
  local volumeMount = k.core.v1.volumeMount,

  local target_name = 'metrics-generator',
  local tempo_config_volume = 'tempo-conf',
  local tempo_data_volume = 'metrics-generator-data',
  local tempo_overrides_config_volume = 'overrides',

  tempo_metrics_generator_ports:: [containerPort.new('prom-metrics', $._config.port)],
  tempo_metrics_generator_args:: {
    target: target_name,
    'config.file': '/conf/tempo.yaml',
    'mem-ballast-size-mbs': $._config.ballast_size_mbs,
  },

  tempo_metrics_generator_pvc::
    pvc.new(tempo_data_volume)
    + pvc.mixin.spec.resources.withRequests({ storage: $._config.metrics_generator.pvc_size })
    + pvc.mixin.spec.withAccessModes(['ReadWriteOnce'])
    + pvc.mixin.spec.withStorageClassName($._config.metrics_generator.pvc_storage_class)
    + pvc.mixin.metadata.withLabels({ app: target_name })
    + pvc.mixin.metadata.withNamespace($._config.namespace),

  tempo_metrics_generator_container::
    container.new(target_name, $._images.tempo) +
    container.withPorts($.tempo_metrics_generator_ports) +
    container.withArgs($.util.mapToFlags($.tempo_metrics_generator_args)) +
    container.withVolumeMounts([
      volumeMount.new(tempo_config_volume, '/conf'),
      volumeMount.new(tempo_data_volume, '/var/tempo'),
      volumeMount.new(tempo_overrides_config_volume, '/overrides'),
    ]) +
    $.util.withResources($._config.metrics_generator.resources) +
    (if $._config.variables_expansion then container.withEnvMixin($._config.variables_expansion_env_mixin) else {}) +
    container.mixin.resources.withRequestsMixin({ 'ephemeral-storage': $._config.metrics_generator.ephemeral_storage_request_size }) +
    container.mixin.resources.withLimitsMixin({ 'ephemeral-storage': $._config.metrics_generator.ephemeral_storage_limit_size }) +
    $.util.readinessProbe +
    (if $._config.variables_expansion then container.withArgsMixin(['-config.expand-env=true']) else {}),

  tempo_metrics_generator_deployment:
    deployment.new(
      target_name,
      0,
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
      volume.fromEmptyDir(tempo_data_volume),
    ])
  ,

  newGeneratorStatefulSet(name, container, with_anti_affinity=true)::
    statefulset.new(
      name,
      $._config.metrics_generator.replicas,
      $.tempo_metrics_generator_container,
      self.tempo_metrics_generator_pvc,
      {
        app: target_name,
        [$._config.gossip_member_label]: 'true',
      },
    )
    + kausal.util.antiAffinityStatefulSet
    + statefulset.mixin.spec.withServiceName(target_name)
    + statefulset.mixin.spec.template.metadata.withAnnotations({
      config_hash: std.md5(std.toString($.tempo_metrics_generator_configmap.data['tempo.yaml'])),
    })
    + statefulset.mixin.spec.template.spec.withVolumes([
      volume.fromConfigMap(tempo_config_volume, $.tempo_metrics_generator_configmap.metadata.name),
      volume.fromConfigMap(tempo_overrides_config_volume, $._config.overrides_configmap_name),
    ]) +
    statefulset.mixin.spec.withPodManagementPolicy('Parallel') +
    $.util.podPriority('high') +
    (if with_anti_affinity then $.util.antiAffinity else {}),

  tempo_metrics_generator_statefulset:
    $.newGeneratorStatefulSet(target_name, self.tempo_metrics_generator_container)
    + statefulset.spec.template.spec.securityContext.withFsGroup(10001)  // 10001 is the UID of the tempo user
    + statefulset.mixin.spec.withReplicas($._config.metrics_generator.replicas),

  tempo_metrics_generator_service:
    kausal.util.serviceFor($.tempo_metrics_generator_deployment),
}
