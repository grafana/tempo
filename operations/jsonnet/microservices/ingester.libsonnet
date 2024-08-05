{
  local k = import 'k.libsonnet',
  local kausal = import 'ksonnet-util/kausal.libsonnet',
  local container = k.core.v1.container,
  local containerPort = kausal.core.v1.containerPort,
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

  tempo_ingester_ports:: [containerPort.new('prom-metrics', $._config.port)],
  tempo_ingester_args:: {
    target: target_name,
    'config.file': '/conf/tempo.yaml',
    'mem-ballast-size-mbs': $._config.ballast_size_mbs,
  },

  tempo_ingester_pvc::
    pvc.new(tempo_data_volume)
    + pvc.mixin.spec.resources.withRequests({ storage: $._config.ingester.pvc_size })
    + pvc.mixin.spec.withAccessModes(['ReadWriteOnce'])
    + pvc.mixin.spec.withStorageClassName($._config.ingester.pvc_storage_class)
    + pvc.mixin.metadata.withLabels({ app: target_name })
    + pvc.mixin.metadata.withNamespace($._config.namespace),

  tempo_ingester_container::
    container.new(target_name, $._images.tempo) +
    container.withPorts($.tempo_ingester_ports) +
    container.withArgs($.util.mapToFlags($.tempo_ingester_args)) +
    (if $._config.variables_expansion then container.withEnvMixin($._config.variables_expansion_env_mixin) else {}) +
    container.withVolumeMounts([
      volumeMount.new(tempo_config_volume, '/conf'),
      volumeMount.new(tempo_data_volume, '/var/tempo'),
      volumeMount.new(tempo_overrides_config_volume, '/overrides'),
    ]) +
    $.util.withResources($._config.ingester.resources) +
    $.util.readinessProbe +
    (if $._config.variables_expansion then container.withArgsMixin(['-config.expand-env=true']) else {}),

  newIngesterStatefulSet(name, container, with_anti_affinity=true)::
    statefulset.new(
      name,
      3,
      container,
      self.tempo_ingester_pvc,
      {
        app: target_name,
        [$._config.gossip_member_label]: 'true',
      },
    )
    + kausal.util.antiAffinityStatefulSet
    + statefulset.mixin.spec.withServiceName(target_name)
    + statefulset.mixin.spec.template.metadata.withAnnotations({
      config_hash: std.md5(std.toString($.tempo_ingester_configmap.data['tempo.yaml'])),
    })
    + statefulset.mixin.spec.template.spec.withVolumes([
      volume.fromConfigMap(tempo_config_volume, $.tempo_ingester_configmap.metadata.name),
      volume.fromConfigMap(tempo_overrides_config_volume, $._config.overrides_configmap_name),
    ]) +
    statefulset.mixin.spec.withPodManagementPolicy('Parallel') +
    statefulset.mixin.spec.template.spec.withTerminationGracePeriodSeconds(1200) +
    $.util.podPriority('high') +
    (if with_anti_affinity then $.util.antiAffinity else {})
  ,

  tempo_ingester_statefulset:
    $.newIngesterStatefulSet(target_name, self.tempo_ingester_container)
    + statefulset.spec.template.spec.securityContext.withFsGroup(10001)  // 10001 is the UID of the tempo user
    + statefulset.mixin.spec.withReplicas($._config.ingester.replicas),

  tempo_ingester_service:
    kausal.util.serviceFor($.tempo_ingester_statefulset),

  local podDisruptionBudget = k.policy.v1.podDisruptionBudget,
  ingester_pdb:
    podDisruptionBudget.new(target_name) +
    podDisruptionBudget.mixin.metadata.withLabels({ name: target_name }) +
    podDisruptionBudget.mixin.spec.selector.withMatchLabels({ name: target_name }) +
    podDisruptionBudget.mixin.spec.withMaxUnavailable(1),
}
