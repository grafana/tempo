{
  local container = $.core.v1.container,
  local deployment = $.apps.v1.deployment,
  local statefulSet = $.apps.v1.statefulSet,
  local configMap = $.core.v1.configMap,
  local podDisruptionBudget = $.policy.v1.podDisruptionBudget,
  local volume = $.core.v1.volume,
  local roleBinding = $.rbac.v1.roleBinding,
  local role = $.rbac.v1.role,
  local service = $.core.v1.service,
  local serviceAccount = $.core.v1.serviceAccount,
  local servicePort = $.core.v1.servicePort,
  local policyRule = $.rbac.v1.policyRule,
  local podAntiAffinity = deployment.mixin.spec.template.spec.affinity.podAntiAffinity,
  local k = import 'ksonnet-util/kausal.libsonnet',
  local containerPort = k.core.v1.containerPort,
  local volumeMount = k.core.v1.volumeMount,
  local pvc = k.core.v1.persistentVolumeClaim,

  //
  // Multi-zone live-stores (non-optional).
  //

  local target_name = 'live-store',
  local tempo_config_volume = 'tempo-conf',
  local tempo_data_volume = 'live-store-data',
  local tempo_overrides_config_volume = 'overrides',

  tempo_live_store_ports:: [containerPort.new('prom-metrics', $._config.port)],
  tempo_live_store_args:: {
    target: target_name,
    'config.file': '/conf/tempo.yaml',
    'mem-ballast-size-mbs': $._config.ballast_size_mbs,
  },

  tempo_live_store_pvc::
    pvc.new(tempo_data_volume)
    + pvc.mixin.spec.resources.withRequests({ storage: $._config.live_store.pvc_size })
    + pvc.mixin.spec.withAccessModes(['ReadWriteOnce'])
    + pvc.mixin.spec.withStorageClassName($._config.live_store.pvc_storage_class)
    + pvc.mixin.metadata.withLabels({ app: target_name })
    + pvc.mixin.metadata.withNamespace($._config.namespace),

  tempo_live_store_container::
    container.new(target_name, $._images.tempo_live_store) +
    container.withPorts($.tempo_live_store_ports) +
    container.withArgs($.util.mapToFlags($.tempo_live_store_args)) +
    (if $._config.variables_expansion then container.withEnvMixin($._config.variables_expansion_env_mixin) else {}) +
    container.withVolumeMounts([
      volumeMount.new(tempo_config_volume, '/conf'),
      volumeMount.new(tempo_data_volume, '/var/tempo/live-store'),
      volumeMount.new(tempo_overrides_config_volume, '/overrides'),
    ]) +
    $.util.withResources($._config.live_store.resources) +
    (if $._config.variables_expansion then container.withEnvMixin($._config.variables_expansion_env_mixin) else {}) +
    $.util.readinessProbe +
    (if $._config.variables_expansion then container.withArgsMixin(['-config.expand-env=true']) else {}),

  tempo_live_store_zone_a_args:: {
    'live-store.instance-availability-zone': 'zone-a',
  },
  tempo_live_store_zone_b_args:: {
    'live-store.instance-availability-zone': 'zone-b',
  },

  newLiveStoreZoneContainer(zone, zone_args)::
    $.tempo_live_store_container +
    container.withArgs($.util.mapToFlags(
      $.tempo_live_store_args + zone_args
    )),

  // Helper function for zone-specific anti-affinity rules
  liveStoreZoneAntiAffinity(zone_name)::
    if $._config.live_store.allow_multiple_replicas_on_same_node then {} else {
      spec+:
        // Allow multiple live-stores from the same zone on one node,
        // but prevent live-stores from different zones on the same node.
        podAntiAffinity.withRequiredDuringSchedulingIgnoredDuringExecution([
          podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecutionType.new() +
          podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecutionType.mixin.labelSelector.withMatchExpressions([
            { key: 'rollout-group', operator: 'In', values: ['live-store'] },
            { key: 'name', operator: 'NotIn', values: [zone_name] },
          ]) +
          podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecutionType.withTopologyKey('kubernetes.io/hostname'),
        ]).spec,
    },

  newLiveStoreZoneStatefulSet(zone, container)::
    local name = 'live-store-zone-%s' % zone;

    self.newLiveStoreStatefulSet(name, container) +
    statefulSet.mixin.metadata.withLabels({ 'rollout-group': 'live-store' }) +
    statefulSet.mixin.metadata.withAnnotations({ 'rollout-max-unavailable': std.toString($._config.live_store.max_unavailable) }) +
    statefulSet.mixin.spec.template.metadata.withLabels({ name: name, 'rollout-group': 'live-store' }) +
    statefulSet.mixin.spec.selector.withMatchLabels({ name: name, 'rollout-group': 'live-store' }) +
    statefulSet.mixin.spec.updateStrategy.withType('OnDelete') +
    statefulSet.mixin.spec.template.spec.withTerminationGracePeriodSeconds(1200) +
    statefulSet.mixin.spec.withReplicas($._config.live_store.replicas) +
    (if !std.isObject($._config.node_selector) then {} else statefulSet.mixin.spec.template.spec.withNodeSelectorMixin($._config.node_selector)) +
    statefulSet.spec.template.spec.securityContext.withFsGroup(10001) +  // 10001 is the UID of the tempo user
    self.liveStoreZoneAntiAffinity(name),

  newLiveStoreStatefulSet(name, container)::
    statefulSet.new(
      name,
      3,
      container,
      self.tempo_live_store_pvc,
      {
        app: target_name,
        [$._config.gossip_member_label]: 'true',
      },
    )
    + statefulSet.mixin.spec.withServiceName(name)
    + statefulSet.mixin.spec.template.metadata.withAnnotations({
      config_hash: std.md5(std.toString($.tempo_live_store_configmap.data['tempo.yaml'])),
    })
    + statefulSet.mixin.spec.template.spec.withVolumes([
      volume.fromConfigMap(tempo_config_volume, $.tempo_live_store_configmap.metadata.name),
      volume.fromConfigMap(tempo_overrides_config_volume, $._config.overrides_configmap_name),
    ]) +
    statefulSet.mixin.spec.withPodManagementPolicy('Parallel') +
    statefulSet.mixin.spec.template.spec.withTerminationGracePeriodSeconds(1200) +
    $.util.podPriority('high'),

  newLiveStoreZoneService(sts)::
    $.util.serviceFor(sts, $._config.service_ignored_labels) +
    service.mixin.spec.withClusterIp('None'),  // Headless.

  tempo_live_store_zone_a_container::
    self.newLiveStoreZoneContainer('a', $.tempo_live_store_zone_a_args),

  tempo_live_store_zone_a_statefulset:
    self.newLiveStoreZoneStatefulSet('a', $.tempo_live_store_zone_a_container),

  tempo_live_store_zone_a_service:
    $.newLiveStoreZoneService($.tempo_live_store_zone_a_statefulset),

  tempo_live_store_zone_b_container::
    self.newLiveStoreZoneContainer('b', $.tempo_live_store_zone_b_args),

  tempo_live_store_zone_b_statefulset:
    self.newLiveStoreZoneStatefulSet('b', $.tempo_live_store_zone_b_container),

  tempo_live_store_zone_b_service:
    $.newLiveStoreZoneService($.tempo_live_store_zone_b_statefulset),

  live_store_rollout_pdb:
    podDisruptionBudget.new('live-store-rollout-pdb') +
    podDisruptionBudget.mixin.metadata.withLabels({ name: 'live-store-rollout-pdb' }) +
    podDisruptionBudget.mixin.spec.selector.withMatchLabels({ 'rollout-group': 'live-store' }) +
    podDisruptionBudget.mixin.spec.withMaxUnavailable(1),

  tempo_live_store_configmap:
    configMap.new('tempo-live-store') +
    configMap.withData({
      'tempo.yaml': k.util.manifestYaml($.tempo_live_store_config),
    }),

  //
  // Multi-zone live-stores are non-optional - no single-zone fallback.
  //

  tempo_live_store_statefulset: null,  // Only multi-zone deployments are supported

  tempo_live_store_service: null,  // Only multi-zone services are supported

  live_store_pdb: null,  // Only rollout PDB is used
}
