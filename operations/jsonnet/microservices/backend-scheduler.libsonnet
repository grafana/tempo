{
  local k = import 'ksonnet-util/kausal.libsonnet',

  local container = k.core.v1.container,
  local containerPort = k.core.v1.containerPort,
  local volumeMount = k.core.v1.volumeMount,
  local statefulset = k.apps.v1.statefulSet,
  local volume = k.core.v1.volume,
  local envVar = k.core.v1.envVar,
  local configMap = k.core.v1.configMap,

  local target_name = 'backend-scheduler',
  local tempo_config_volume = 'tempo-conf',
  local tempo_overrides_config_volume = 'overrides',

  // Statefulset

  tempo_backend_scheduler_ports:: [
    containerPort.new('prom-metrics', $._config.port),
  ],

  tempo_backend_scheduler_args:: {
    target: target_name,
    'config.file': '/conf/tempo.yaml',
    'mem-ballast-size-mbs': $._config.ballast_size_mbs,
  },

  tempo_backend_scheduler_container::
    container.new(target_name, $._images.tempo) +
    container.withPorts($.tempo_backend_scheduler_ports) +
    container.withArgs($.util.mapToFlags($.tempo_backend_scheduler_args)) +
    container.withVolumeMounts([
      volumeMount.new(tempo_config_volume, '/conf'),
      volumeMount.new(tempo_overrides_config_volume, '/overrides'),
    ]) +
    $.util.withResources($._config.backend_scheduler.resources) +
    (if $._config.variables_expansion then container.withEnvMixin($._config.variables_expansion_env_mixin) else {}) +
    $.util.readinessProbe +
    (if $._config.variables_expansion then container.withArgsMixin(['-config.expand-env=true']) else {}),

  newBackendSchedulerStatefulSet(max_unavailable=1)::
    statefulset.new(target_name, $._config.backend_scheduler.replicas, $.tempo_backend_scheduler_container, [], { app: target_name }) +
    statefulset.mixin.spec.withServiceName(target_name) +
    statefulset.spec.template.spec.securityContext.withFsGroup(10001) +  // 10001 is the UID of the tempo user
    statefulset.mixin.spec.template.metadata.withAnnotations({
      config_hash: std.md5(std.toString($.tempo_backend_scheduler_configmap.data['tempo.yaml'])),
    }) +
    statefulset.mixin.spec.template.spec.withVolumes([
      volume.fromConfigMap(tempo_config_volume, $.tempo_backend_scheduler_configmap.metadata.name),
      volume.fromConfigMap(tempo_overrides_config_volume, $._config.overrides_configmap_name),
    ]) +
    statefulset.mixin.spec.withPodManagementPolicy('Parallel'),

  tempo_backend_scheduler_statefulset:
    $.newBackendSchedulerStatefulSet(),

  // Configmap

  tempo_backend_scheduler_configmap:
    configMap.new('tempo-backend-scheduler') +
    configMap.withData({
      'tempo.yaml': k.util.manifestYaml($.tempo_backend_scheduler_config),
    }),

  // Service

  local service = k.core.v1.service,
  local servicePort = k.core.v1.servicePort,
  tempo_backend_scheduler_service:
    k.util.serviceFor($.tempo_backend_scheduler_statefulset)
    + service.mixin.spec.withPortsMixin([
      servicePort.withName('grpc')
      + servicePort.withPort(9095)
      + servicePort.withTargetPort(9095),
    ]),

}
