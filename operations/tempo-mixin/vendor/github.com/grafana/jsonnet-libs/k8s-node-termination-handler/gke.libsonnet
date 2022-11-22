local k = import 'k.libsonnet';

{
  namespace:: 'kube-system',
  slack_webhook:: '',
  image:: 'k8s.gcr.io/gke-node-termination-handler@sha256:aca12d17b222dfed755e28a44d92721e477915fb73211d0a0f8925a1fa847cca',

  local container = k.core.v1.container,
  local envVar = k.core.v1.envVar,
  container::
    container.new('node-termination-handler', self.image)
    + container.withCommand(['./node-termination-handler'])
    + container.withArgs([
      '--logtostderr',
      '--exclude-pods=$(POD_NAME):$(POD_NAMESPACE)',
      '-v=10',
      '--taint=cloud.google.com/impending-node-termination::NoSchedule',
    ])
    + container.withEnv([
      envVar.fromFieldPath('POD_NAME', 'metadata.name'),
      envVar.fromFieldPath('POD_NAMESPACE', 'metadata.namespace'),
      envVar.new('SLACK_WEBHOOK_URL', self.slack_webhook),
    ])
    + container.securityContext.capabilities.withAdd(['SYS_BOOT'])
    + container.resources.withRequests({
      cpu: '50m',
      memory: '10Mi',
    })
    + container.resources.withLimits({
      cpu: '150m',
      memory: '30Mi',
    })
  ,

  local daemonSet = k.apps.v1.daemonSet,
  local tolerations = k.core.v1.toleration,
  local nodeAffinity = daemonSet.spec.template.spec.affinity.nodeAffinity,
  local nodeSelectorTerm = k.core.v1.nodeSelectorTerm,
  local nodeSelectorRequirement = k.core.v1.nodeSelectorRequirement,
  daemonset:
    local labels = { name: 'node-termination-handler' };
    daemonSet.new(labels.name)
    + daemonSet.metadata.withNamespace(self.namespace)
    + daemonSet.metadata.withLabels(labels)
    + daemonSet.spec.withMinReadySeconds(10)
    + daemonSet.spec.selector.withMatchLabels(labels)
    + daemonSet.spec.template.metadata.withLabels(labels)
    + daemonSet.spec.template.spec.withContainers([self.container])
    + daemonSet.spec.template.spec.withServiceAccount(self.service_account.metadata.name)
    + daemonSet.spec.template.spec.withHostPID(true)
    + daemonSet.spec.template.spec.withTolerations([
      tolerations.withOperator('Exists')
      + tolerations.withEffect('NoSchedule'),
      tolerations.withOperator('Exists')
      + tolerations.withEffect('NoExecute'),
    ])
    + nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.withNodeSelectorTerms(
      [
        nodeSelectorTerm.withMatchExpressions([
          nodeSelectorRequirement.withKey('cloud.google.com/gke-accelerator')
          + nodeSelectorRequirement.withOperator('Exists'),
        ]),
        nodeSelectorTerm.withMatchExpressions([
          nodeSelectorRequirement.withKey('cloud.google.com/gke-preemptible')
          + nodeSelectorRequirement.withOperator('Exists'),
        ]),
      ]
    )
  ,

  local serviceAccount = k.core.v1.serviceAccount,
  service_account:
    serviceAccount.new('node-termination-handler')
    + serviceAccount.metadata.withNamespace(self.namespace)
  ,

  local clusterRole = k.rbac.v1.clusterRole,
  local policyRule = k.rbac.v1.policyRule,
  cluster_role:
    clusterRole.new('node-termination-handler-role')
    + clusterRole.withRules([
      policyRule.withApiGroups('')
      + policyRule.withResources(['nodes'])
      + policyRule.withVerbs(['get', 'update'])
      ,
      policyRule.withApiGroups('')
      + policyRule.withResources(['events'])
      + policyRule.withVerbs(['create'])
      ,
      policyRule.withApiGroups('')
      + policyRule.withResources(['pods'])
      + policyRule.withVerbs(['get', 'list', 'delete']),
    ])
  ,

  local clusterRoleBinding = k.rbac.v1.clusterRoleBinding,
  local subject = k.rbac.v1.subject,
  cluster_role_binding:
    clusterRoleBinding.new('node-termination-handler-role-binding')
    + clusterRoleBinding.roleRef.withApiGroup('rbac.authorization.k8s.io')
    + clusterRoleBinding.roleRef.withKind('ClusterRole')
    + clusterRoleBinding.roleRef.withName('node-termination-handler-role')
    + clusterRoleBinding.withSubjects(
      subject.withKind(self.service_account.kind)
      + subject.withName(self.service_account.metadata.name)
      + subject.withNamespace(self.service_account.metadata.namespace)
      ,
    ),
}
