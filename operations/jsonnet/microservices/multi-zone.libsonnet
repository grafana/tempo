{
  local container = $.core.v1.container,
  local deployment = $.apps.v1.deployment,
  local statefulSet = $.apps.v1.statefulSet,
  local podDisruptionBudget = $.policy.v1.podDisruptionBudget,
  local volume = $.core.v1.volume,
  local roleBinding = $.rbac.v1.roleBinding,
  local role = $.rbac.v1.role,
  local service = $.core.v1.service,
  local serviceAccount = $.core.v1.serviceAccount,
  local servicePort = $.core.v1.servicePort,
  local policyRule = $.rbac.v1.policyRule,
  local podAntiAffinity = deployment.mixin.spec.template.spec.affinity.podAntiAffinity,

  _config+: {
    multi_zone_ingester_enabled: false,
    multi_zone_ingester_migration_enabled: false,
    multi_zone_ingester_replicas: 0,
    multi_zone_ingester_max_unavailable: 25,
  },

  tempo_config+: {
    ingester+: {
      lifecycler+: {
        ring+: (if $._config.multi_zone_ingester_enabled then { zone_awareness_enabled: $._config.multi_zone_ingester_enabled } else {}),
      } + (if $._config.multi_zone_ingester_enabled then { availability_zone: '${AVAILABILITY_ZONE}' } else {}),
    },
  },

  //
  // Multi-zone ingesters.
  //

  tempo_ingester_zone_a_args:: {},
  tempo_ingester_zone_b_args:: {},
  tempo_ingester_zone_c_args:: {},

  newIngesterZoneContainer(zone, zone_args)::
    local zone_name = 'zone-%s' % zone;

    $.tempo_ingester_container +
    container.withArgs($.util.mapToFlags(
      $.tempo_ingester_args + zone_args
    )) +
    container.withEnvMixin([{ name: 'AVAILABILITY_ZONE', value: zone_name }]) +
    (if $._config.variables_expansion then container.withArgsMixin(['-config.expand-env=true']) else {}),

  newIngesterZoneStatefulSet(zone, container)::
    local name = 'ingester-zone-%s' % zone;

    self.newIngesterStatefulSet(name, container, with_anti_affinity=false) +
    statefulSet.mixin.metadata.withLabels({ 'rollout-group': 'ingester' }) +
    statefulSet.mixin.metadata.withAnnotations({ 'rollout-max-unavailable': std.toString($._config.multi_zone_ingester_max_unavailable) }) +
    statefulSet.mixin.spec.template.metadata.withLabels({ name: name, 'rollout-group': 'ingester' }) +
    statefulSet.mixin.spec.selector.withMatchLabels({ name: name, 'rollout-group': 'ingester' }) +
    statefulSet.mixin.spec.updateStrategy.withType('OnDelete') +
    statefulSet.mixin.spec.template.spec.withTerminationGracePeriodSeconds(1200) +
    statefulSet.mixin.spec.withReplicas(std.ceil($._config.multi_zone_ingester_replicas / 3)) +
    (if !std.isObject($._config.node_selector) then {} else statefulSet.mixin.spec.template.spec.withNodeSelectorMixin($._config.node_selector)) +
    if $._config.ingester_allow_multiple_replicas_on_same_node then {} else {
      spec+:
        // Allow to schedule 2+ ingesters in the same zone on the same node, but do not schedule 2+ ingesters in
        // different zones on the same node. In case of 1 node failure in the Kubernetes cluster, only ingesters
        // in 1 zone will be affected.
        podAntiAffinity.withRequiredDuringSchedulingIgnoredDuringExecution([
          podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecutionType.new() +
          podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecutionType.mixin.labelSelector.withMatchExpressions([
            { key: 'rollout-group', operator: 'In', values: ['ingester'] },
            { key: 'name', operator: 'NotIn', values: [name] },
          ]) +
          podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecutionType.withTopologyKey('kubernetes.io/hostname'),
        ]).spec,
    },

  // Creates a headless service for the per-zone ingesters StatefulSet. We don't use it
  // but we need to create it anyway because it's responsible for the network identity of
  // the StatefulSet pods. For more information, see:
  // https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#statefulset-v1-apps
  newIngesterZoneService(sts)::
    $.util.serviceFor(sts, $._config.service_ignored_labels) +
    service.mixin.spec.withClusterIp('None'),  // Headless.

  tempo_ingester_zone_a_container:: if !$._config.multi_zone_ingester_enabled then null else
    self.newIngesterZoneContainer('a', $.tempo_ingester_zone_a_args),

  tempo_ingester_zone_a_statefulset: if !$._config.multi_zone_ingester_enabled then null else
    self.newIngesterZoneStatefulSet('a', $.tempo_ingester_zone_a_container),

  tempo_ingester_zone_a_service: if !$._config.multi_zone_ingester_enabled then null else
    $.newIngesterZoneService($.tempo_ingester_zone_a_statefulset),

  tempo_ingester_zone_b_container:: if !$._config.multi_zone_ingester_enabled then null else
    self.newIngesterZoneContainer('b', $.tempo_ingester_zone_b_args),

  tempo_ingester_zone_b_statefulset: if !$._config.multi_zone_ingester_enabled then null else
    self.newIngesterZoneStatefulSet('b', $.tempo_ingester_zone_b_container),

  tempo_ingester_zone_b_service: if !$._config.multi_zone_ingester_enabled then null else
    $.newIngesterZoneService($.tempo_ingester_zone_b_statefulset),

  tempo_ingester_zone_c_container:: if !$._config.multi_zone_ingester_enabled then null else
    self.newIngesterZoneContainer('c', $.tempo_ingester_zone_c_args),

  tempo_ingester_zone_c_statefulset: if !$._config.multi_zone_ingester_enabled then null else
    self.newIngesterZoneStatefulSet('c', $.tempo_ingester_zone_c_container),

  tempo_ingester_zone_c_service: if !$._config.multi_zone_ingester_enabled then null else
    $.newIngesterZoneService($.tempo_ingester_zone_c_statefulset),

  ingester_rollout_pdb: if !$._config.multi_zone_ingester_enabled then null else
    podDisruptionBudget.new('ingester-rollout-pdb') +
    podDisruptionBudget.mixin.metadata.withLabels({ name: 'ingester-rollout-pdb' }) +
    podDisruptionBudget.mixin.spec.selector.withMatchLabels({ 'rollout-group': 'ingester' }) +
    podDisruptionBudget.mixin.spec.withMaxUnavailable(1),

  //
  // Single-zone ingesters shouldn't be configured when multi-zone is enabled.
  //

  tempo_ingester_statefulset:
    // Remove the default "ingester" StatefulSet if multi-zone is enabled and no migration is in progress.
    if $._config.multi_zone_ingester_enabled && !$._config.multi_zone_ingester_migration_enabled
    then null
    else super.tempo_ingester_statefulset,

  tempo_ingester_service:
    // Remove the default "ingester" service if multi-zone is enabled and no migration is in progress.
    if $._config.multi_zone_ingester_enabled && !$._config.multi_zone_ingester_migration_enabled
    then null
    else super.tempo_ingester_service,

  ingester_pdb:
    // Keep it if multi-zone is disabled.
    if !$._config.multi_zone_ingester_enabled
    then super.ingester_pdb
    // We don't want Kubernetes to terminate any "ingester" StatefulSet's pod while migration is in progress.
    else if $._config.multi_zone_ingester_migration_enabled
    then super.ingester_pdb + podDisruptionBudget.mixin.spec.withMaxUnavailable(0)
    // Remove it if multi-zone is enabled and no migration is in progress.
    else null,
}
