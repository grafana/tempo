{
  local k = import 'ksonnet-util/kausal.libsonnet',

  local container = k.core.v1.container,
  local deployment = k.apps.v1.deployment,
  local policyRule = k.rbac.v1.policyRule,
  local role = k.rbac.v1.role,
  local roleBinding = k.rbac.v1.roleBinding,
  local serviceAccount = k.core.v1.serviceAccount,
  local podDisruptionBudget = k.policy.v1.podDisruptionBudget,

  _config+:: {
    rollout_operator_enabled: $._config.block_builder_concurrent_rollout_enabled,
  },

  rollout_operator_args:: {
    'kubernetes.namespace': $._config.namespace,
  },

  rollout_operator_container::
    container.new('rollout-operator', $._images.rollout_operator) +
    container.withArgsMixin($.util.mapToFlags($.rollout_operator_args)) +
    container.withPorts([$.core.v1.containerPort.new('http-metrics', 8001)]) +
    $.util.resourcesRequests('100m', '100Mi') +
    $.util.resourcesLimits(null, '200Mi') +
    container.mixin.readinessProbe.httpGet.withPath('/ready') +
    container.mixin.readinessProbe.httpGet.withPort(8001) +
    container.mixin.readinessProbe.withInitialDelaySeconds(5) +
    container.mixin.readinessProbe.withTimeoutSeconds(1),

  rollout_operator_deployment: if !$._config.rollout_operator_enabled then null else
    deployment.new('rollout-operator', 1, [$.rollout_operator_container]) +
    deployment.mixin.metadata.withName('rollout-operator') +
    deployment.mixin.spec.template.spec.withServiceAccountName('rollout-operator') +
    deployment.mixin.spec.strategy.rollingUpdate.withMaxSurge(0) +
    deployment.mixin.spec.strategy.rollingUpdate.withMaxUnavailable(1),

  rollout_operator_role: if !$._config.rollout_operator_enabled then null else
    role.new('rollout-operator-role') +
    role.mixin.metadata.withNamespace($._config.namespace) +
    role.withRulesMixin([
      policyRule.withApiGroups('') +
      policyRule.withResources(['pods']) +
      policyRule.withVerbs(['list', 'get', 'watch', 'delete']),
      policyRule.withApiGroups('apps') +
      policyRule.withResources(['statefulsets']) +
      policyRule.withVerbs(['list', 'get', 'watch', 'patch']),
      policyRule.withApiGroups('apps') +
      policyRule.withResources(['statefulsets/status']) +
      policyRule.withVerbs(['update']),
    ]),

  rollout_operator_rolebinding: if !$._config.rollout_operator_enabled then null else
    roleBinding.new('rollout-operator-rolebinding') +
    roleBinding.mixin.metadata.withNamespace($._config.namespace) +
    roleBinding.mixin.roleRef.withApiGroup('rbac.authorization.k8s.io') +
    roleBinding.mixin.roleRef.withKind('Role') +
    roleBinding.mixin.roleRef.withName('rollout-operator-role') +
    roleBinding.withSubjectsMixin({
      kind: 'ServiceAccount',
      name: 'rollout-operator',
      namespace: $._config.namespace,
    }),

  rollout_operator_service_account: if !$._config.rollout_operator_enabled then null else
    serviceAccount.new('rollout-operator'),

  rollout_operator_pdb: if !$._config.rollout_operator_enabled then null else
    podDisruptionBudget.new('rollout-operator') +
    podDisruptionBudget.mixin.metadata.withLabels({ name: 'rollout-operator' }) +
    podDisruptionBudget.mixin.spec.selector.withMatchLabels({ name: 'rollout-operator' }) +
    podDisruptionBudget.mixin.spec.withMaxUnavailable(1),
}
