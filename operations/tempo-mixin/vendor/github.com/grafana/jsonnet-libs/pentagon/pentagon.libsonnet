local kausal = import 'ksonnet-util/kausal.libsonnet';

(import 'config.libsonnet') +
{
  local this = self,
  local k = kausal { _config+:: this._config },

  local container = k.core.v1.container,
  local containerPort = k.core.v1.containerPort,
  container::
    container.new('pentagon', this._images.pentagon)
    + container.withImagePullPolicy('IfNotPresent')
    + container.withArgs([
      '/etc/pentagon/pentagon.yaml',
    ])
    + container.withPorts([
      containerPort.newNamed(name='http-metrics', containerPort=8888),
    ])
    + k.util.resourcesRequests('100m', '200Mi')
    + k.util.resourcesLimits('200m', '400Mi'),

  local deployment = k.apps.v1.deployment,
  deployment:
    deployment.new(
      '%(name)s-%(vault_role)s' % this._config.pentagon,
      1,
      [this.container]
    ) +
    deployment.spec.template.spec.withServiceAccountName(this.rbac.service_account.metadata.name) +
    k.util.configMapVolumeMount(this.config_map, '/etc/pentagon'),

  withVaultCertFile(filename, configmap_name, hash=''):: {
    pentagonConfig+:: {
      vault+: {
        tls+: {
          // This needs to directly refer to a cert with `cacert`,
          // using `capath` requires a run of c_rehash,
          // which is much more complicated to implement
          cacert: '/ssl/%s' % filename,
        },
      },
    },
    deployment+:
      k.util.configVolumeMount(configmap_name, '/ssl')
      + (
        if hash != ''
        then deployment.spec.template.metadata.withAnnotationsMixin({
          ['%s-hash' % configmap_name]: hash,
        })
        else {}
      ),
  },

  local configMap = k.core.v1.configMap,
  config_map:
    configMap.new('%(name)s-%(vault_role)s' % this._config.pentagon)
    + configMap.withData({
      'pentagon.yaml': k.util.manifestYaml(this.pentagonConfig),
    }),

  // rbac.service_account is used by pentagon for authenticating against Vault
  // and updating k8s secrets.
  local policyRule = k.rbac.v1.policyRule,
  rbac:
    k.util.namespacedRBAC(
      this._config.pentagon.name,
      [
        policyRule.withApiGroups('')
        + policyRule.withResources(['secrets'])
        + policyRule.withVerbs(['get', 'list', 'watch', 'update', 'create', 'delete']),
      ]
    ),

  local clusterRole = k.rbac.v1.clusterRole,
  // clusterRole is needed to allow Vault to verify the token.
  cluster_role:
    clusterRole.new('%s-%s' % [this._config.pentagon.name, this._config.namespace])
    + clusterRole.withRulesMixin([
      policyRule.withApiGroups('authentication.k8s.io')
      + policyRule.withResources(['tokenreviews'])
      + policyRule.withVerbs(['create']),

      policyRule.withApiGroups('authentication.k8s.io')
      + policyRule.withResources(['subjectaccessreviews'])
      + policyRule.withVerbs(['create']),
    ]),

  local clusterRoleBinding = k.rbac.v1.clusterRoleBinding,
  cluster_role_binding:
    clusterRoleBinding.new('%s-%s' % [this._config.pentagon.name, this._config.namespace])
    + clusterRoleBinding.roleRef.withApiGroup('rbac.authorization.k8s.io')
    + clusterRoleBinding.roleRef.withKind('ClusterRole')
    + clusterRoleBinding.roleRef.withName(this.cluster_role.metadata.name)
    + clusterRoleBinding.withSubjectsMixin({
      kind: 'ServiceAccount',
      name: this.rbac.service_account.metadata.name,
      namespace: this.rbac.service_account.metadata.namespace,
    }),
}
