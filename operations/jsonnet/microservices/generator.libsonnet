{
  local k = import 'ksonnet-util/kausal.libsonnet',

  local container = k.core.v1.container,
  local containerPort = k.core.v1.containerPort,
  local deployment = k.apps.v1.deployment,
  local volume = k.core.v1.volume,
  local volumeMount = k.core.v1.volumeMount,

  local target_name = 'metrics-generator',
  local tempo_config_volume = 'tempo-conf',
  local tempo_generator_wal_volume = 'metrics-generator-wal-data',
  local tempo_overrides_config_volume = 'overrides',

  tempo_metrics_generator_ports:: [containerPort.new('prom-metrics', $._config.port)],
  tempo_metrics_generator_args:: {
    target: target_name,
    'config.file': '/conf/tempo.yaml',
    'mem-ballast-size-mbs': $._config.ballast_size_mbs,
  },

  tempo_metrics_generator_container::
    container.new(target_name, $._images.tempo) +
    container.withPorts($.tempo_metrics_generator_ports) +
    container.withArgs($.util.mapToFlags($.tempo_metrics_generator_args)) +
    container.withVolumeMounts([
      volumeMount.new(tempo_config_volume, '/conf'),
      volumeMount.new(tempo_generator_wal_volume, $.tempo_metrics_generator_config.metrics_generator.storage.path),
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
      volume.fromEmptyDir(tempo_generator_wal_volume),
    ]),

  tempo_metrics_generator_service:
    k.util.serviceFor($.tempo_metrics_generator_deployment),
}
