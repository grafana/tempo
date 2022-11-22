{
  _images+:: {
    operator: 'quay.io/coreos/etcd-operator:v0.9.4',
  },

  local kausal = import 'ksonnet-util/kausal.libsonnet',
  local k = kausal { _config+:: $._config },
  local policyRule = k.rbac.v1.policyRule,

  operator_rbac:
    k.util.rbac('etcd-operator', [
      policyRule.new() +
      policyRule.withApiGroups(['etcd.database.coreos.com']) +
      policyRule.withResources(['etcdclusters', 'etcdbackups', 'etcdrestores']) +
      policyRule.withVerbs(['*']),

      policyRule.new() +
      policyRule.withApiGroups(['apiextensions.k8s.io']) +
      policyRule.withResources(['customresourcedefinitions']) +
      policyRule.withVerbs(['*']),

      policyRule.new() +
      policyRule.withApiGroups(['']) +
      policyRule.withResources(['pods', 'services', 'endpoints', 'persistentvolumeclaims', 'events']) +
      policyRule.withVerbs(['*']),

      policyRule.new() +
      policyRule.withApiGroups(['apps']) +
      policyRule.withResources(['deployments']) +
      policyRule.withVerbs(['*']),
    ]),

  local container = k.core.v1.container,
  local env = container.envType,
  operator_container::
    container.new('operator', $._images.operator) +
    container.withCommand(['etcd-operator']) +
    container.withArgs(['-cluster-wide']) +
    container.withEnvMixin([
      env.fromFieldPath('MY_POD_NAMESPACE', 'metadata.namespace'),
      env.fromFieldPath('MY_POD_NAME', 'metadata.name'),
    ]) +
    container.withPorts([k.core.v1.containerPort.new('http-metrics', 8080)]) +
    k.util.resourcesRequests('500m', '200Mi') +
    k.util.resourcesLimits('1', '500Mi'),

  local deployment = k.apps.v1.deployment,

  operator_deployment:
    deployment.new('etcd-operator', 1, [$.operator_container]) +
    deployment.mixin.spec.template.spec.withServiceAccount('etcd-operator'),
}
