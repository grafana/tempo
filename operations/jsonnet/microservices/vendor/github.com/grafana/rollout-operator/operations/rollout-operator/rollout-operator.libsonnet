(import 'ksonnet-util/kausal.libsonnet') +

(import 'images.libsonnet') +

(import 'rollout-operator-utils.libsonnet') +

// Support for ReplicaTemplate objects.
(import 'replica-template.libsonnet') +

// Support for ZoneAwarePodDisruptionBudget objects.
(import 'zone-aware-pod-disruption-budget.libsonnet') +

{
  local clusterRole = $.rbac.v1.clusterRole,
  local clusterRoleBinding = $.rbac.v1.clusterRoleBinding,
  local container = $.core.v1.container,
  local deployment = $.apps.v1.deployment,
  local mutatingWebhook = $.admissionregistration.v1.mutatingWebhook,
  local mutatingWebhookConfiguration = $.admissionregistration.v1.mutatingWebhookConfiguration,
  local policyRule = $.rbac.v1.policyRule,
  local role = $.rbac.v1.role,
  local roleBinding = $.rbac.v1.roleBinding,
  local service = $.core.v1.service,
  local serviceAccount = $.core.v1.serviceAccount,
  local servicePort = $.core.v1.servicePort,
  local validatingWebhook = $.admissionregistration.v1.validatingWebhook,
  local validatingWebhookConfiguration = $.admissionregistration.v1.validatingWebhookConfiguration,

  _config+:: {
    rollout_operator_enabled: true,

    // Configure the rollout operator to accept webhook requests made as part of scaling
    // statefulsets up or down. This allows the rollout operator to ensure that stateful
    // components (ingesters, store-gateways) are scaled up or down safely.
    // This will also enable the zone aware pod disruption budget capability.
    rollout_operator_webhooks_enabled: true,

    // Include the custom resource definiton for the zpdb.
    zpdb_custom_resource_definition_enabled: $._config.rollout_operator_webhooks_enabled,

    // Configure the rollout operator to enable support for ReplicaTemplates
    rollout_operator_replica_template_access_enabled: false,

    // Include the custom resource definiton for ReplicaTemplates.
    replica_template_custom_resource_definition_enabled: $._config.rollout_operator_webhooks_enabled,

    // Ignore_rollout_operator_*_webhook_failures will set the rollout-operator to ignore
    // webhook failures. Useful during a rollout to a new cell, where rollout-operator service
    // is still not created, as the webhook might be created before the service, and that could
    // block other operations that would block the service creation.
    ignore_rollout_operator_no_downscale_webhook_failures: false,
    ignore_rollout_operator_prepare_downscale_webhook_failures: false,
    ignore_rollout_operator_zpdb_eviction_webhook_failures: false,
    ignore_rollout_operator_zpdb_validation_webhook_failures: false,
  },

  assert !$._config.rollout_operator_replica_template_access_enabled || $._config.rollout_operator_webhooks_enabled : 'rollout_operator_replica_template_access_enabled requires rollout_operator_webhooks_enabled=true',
  assert !$._config.rollout_operator_replica_template_access_enabled || $._config.rollout_operator_enabled : 'rollout_operator_replica_template_access_enabled requires rollout_operator_enabled=true',
  assert !$._config.rollout_operator_webhooks_enabled || $._config.rollout_operator_enabled : 'rollout_operator_webhooks_enabled requires rollout_operator_enabled=true',
  assert !$._config.ignore_rollout_operator_no_downscale_webhook_failures || $._config.rollout_operator_webhooks_enabled : 'ignore_rollout_operator_no_downscale_webhook_failures requires rollout_operator_webhooks_enabled=true',
  assert !$._config.ignore_rollout_operator_prepare_downscale_webhook_failures || $._config.rollout_operator_webhooks_enabled : 'ignore_rollout_operator_prepare_downscale_webhook_failures requires rollout_operator_webhooks_enabled=true',
  assert !$._config.ignore_rollout_operator_zpdb_eviction_webhook_failures || $._config.rollout_operator_webhooks_enabled : 'ignore_rollout_operator_zpdb_eviction_webhook_failures requires rollout_operator_webhooks_enabled=true',
  assert !$._config.ignore_rollout_operator_zpdb_validation_webhook_failures || $._config.rollout_operator_webhooks_enabled : 'ignore_rollout_operator_zpdb_validation_webhook_failures requires rollout_operator_webhooks_enabled=true',
  assert !$._config.replica_template_custom_resource_definition_enabled || $._config.rollout_operator_webhooks_enabled : 'replica_template_custom_resource_definition_enabled requires rollout_operator_webhooks_enabled=true',

  local enableWebhooks = $._config.rollout_operator_replica_template_access_enabled || $._config.rollout_operator_webhooks_enabled,

  zpdb_template:: std.parseYaml(importstr 'crds/zone-aware-pod-disruption-budget.yaml'),
  zpdb_custom_resource: if !$._config.zpdb_custom_resource_definition_enabled then null else $.zpdb_template,

  replica_template:: std.parseYaml(importstr 'crds/replica-templates.yaml'),
  replica_template_custom_resource: if !$._config.replica_template_custom_resource_definition_enabled then null else $.replica_template,

  rollout_operator_args:: {
    'kubernetes.namespace': $._config.namespace,
    'use-zone-tracker': true,
    'zone-tracker.config-map-name': 'rollout-operator-zone-tracker',
  } + if enableWebhooks then {
    'server-tls.enabled': 'true',
  } else {},

  rollout_operator_node_affinity_matchers:: [],

  rollout_operator_container::
    container.new('rollout-operator', $._images.rollout_operator) +
    container.withArgsMixin($.util.mapToFlags($.rollout_operator_args)) +
    container.withPorts(
      [$.core.v1.containerPort.new('http-metrics', 8001)] +
      if enableWebhooks then
        [$.core.v1.containerPort.new('https', 8443)]
      else []
    ) +
    $.util.resourcesRequests('100m', '100Mi') +
    $.util.resourcesLimits(null, '200Mi') +
    container.mixin.readinessProbe.httpGet.withPath('/ready') +
    container.mixin.readinessProbe.httpGet.withPort(8001) +
    container.mixin.readinessProbe.withInitialDelaySeconds(5) +
    container.mixin.readinessProbe.withTimeoutSeconds(1),

  rollout_operator_deployment: if !$._config.rollout_operator_enabled then null else
    deployment.new('rollout-operator', 1, [$.rollout_operator_container]) +
    deployment.mixin.metadata.withName('rollout-operator') +
    deployment.mixin.metadata.withNamespace($._config.namespace) +
    deployment.mixin.spec.template.spec.withServiceAccountName('rollout-operator') +
    // Ensure Kubernetes doesn't run 2 operators at the same time.
    deployment.mixin.spec.strategy.rollingUpdate.withMaxSurge(0) +
    deployment.mixin.spec.strategy.rollingUpdate.withMaxUnavailable(1) +
    $.newRolloutOperatorNodeAffinityMatchers($.rollout_operator_node_affinity_matchers),

  rollout_operator_service: if !enableWebhooks then null else
    service.new(
      'rollout-operator',
      { name: 'rollout-operator' },
      servicePort.newNamed('https', 443, 8443) +
      servicePort.withProtocol('TCP'),
    )
    + service.mixin.metadata.withNamespace($._config.namespace),

  rollout_operator_role: if !$._config.rollout_operator_enabled then null else
    role.new('rollout-operator-role') +
    role.mixin.metadata.withNamespace($._config.namespace) +
    role.withRulesMixin(
      [
        policyRule.withApiGroups('') +
        policyRule.withResources(['pods']) +
        policyRule.withVerbs(['list', 'get', 'watch', 'delete']),
        policyRule.withApiGroups('apps') +
        policyRule.withResources(['statefulsets']) +
        policyRule.withVerbs(['list', 'get', 'watch', 'patch']),
        policyRule.withApiGroups('apps') +
        policyRule.withResources(['statefulsets/status']) +
        policyRule.withVerbs(['update']),
        policyRule.withApiGroups('') +
        policyRule.withResources(['configmaps']) +
        policyRule.withVerbs(['get', 'update', 'create']),
      ] +
      (
        if $._config.rollout_operator_replica_template_access_enabled then [
          policyRule.withApiGroups($.replica_template.spec.group) +
          policyRule.withResources(['%s/scale' % $.replica_template.spec.names.plural, '%s/status' % $.replica_template.spec.names.plural]) +
          policyRule.withVerbs(['get', 'patch']),
        ] else []
      ) +
      [
        policyRule.withApiGroups($.zpdb_template.spec.group) +
        policyRule.withResources([$.zpdb_template.spec.names.plural]) +
        policyRule.withVerbs(['get', 'list', 'watch']),
      ]

    ),

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

  rollout_operator_webhook_cert_secret_role: if !enableWebhooks then null else
    role.new('rollout-operator-webhook-cert-secret-role') +
    role.mixin.metadata.withNamespace($._config.namespace) +
    role.withRulesMixin([
      policyRule.withApiGroups('')
      + policyRule.withResources(['secrets'])
      + policyRule.withVerbs(['create']),
      policyRule.withApiGroups('')
      + policyRule.withResources(['secrets'])
      + policyRule.withVerbs(['update', 'get'])
      + policyRule.withResourceNames(['rollout-operator-self-signed-certificate']),
    ]),

  rollout_operator_webhook_cert_secret_rolebinding: if !enableWebhooks then null else
    roleBinding.new('rollout-operator-webhook-cert-secret-rolebinding') +
    roleBinding.mixin.metadata.withNamespace($._config.namespace) +
    roleBinding.mixin.roleRef.withApiGroup('rbac.authorization.k8s.io') +
    roleBinding.mixin.roleRef.withKind('Role') +
    roleBinding.mixin.roleRef.withName('rollout-operator-webhook-cert-secret-role') +
    roleBinding.withSubjectsMixin({
      kind: 'ServiceAccount',
      name: 'rollout-operator',
      namespace: $._config.namespace,
    }),

  rollout_operator_webhook_cert_update_clusterrole: if !enableWebhooks then null else
    clusterRole.new('rollout-operator-%s-webhook-cert-update-role' % $._config.namespace) +
    clusterRole.withRulesMixin([
      policyRule.withApiGroups('admissionregistration.k8s.io')
      + policyRule.withResources(['validatingwebhookconfigurations', 'mutatingwebhookconfigurations'])
      + policyRule.withVerbs(['list', 'patch', 'watch']),
    ]),

  rollout_operator_webhook_cert_update_clusterrolebinding: if !enableWebhooks then null else
    clusterRoleBinding.new('rollout-operator-%s-webhook-cert-secret-rolebinding' % $._config.namespace) +
    clusterRoleBinding.mixin.roleRef.withApiGroup('rbac.authorization.k8s.io') +
    clusterRoleBinding.mixin.roleRef.withKind('ClusterRole') +
    clusterRoleBinding.mixin.roleRef.withName('rollout-operator-%s-webhook-cert-update-role' % $._config.namespace) +
    clusterRoleBinding.withSubjectsMixin({
      kind: 'ServiceAccount',
      name: 'rollout-operator',
      namespace: $._config.namespace,
    }),

  zpdb_validation_webhook: if !enableWebhooks then null else
    validatingWebhookConfiguration.new('zpdb-validation-%s' % $._config.namespace) +
    validatingWebhookConfiguration.mixin.metadata.withLabels({
      'grafana.com/namespace': $._config.namespace,
      'grafana.com/inject-rollout-operator-ca': 'true',
    }) +
    validatingWebhookConfiguration.withWebhooksMixin([
      validatingWebhook.withName('zpdb-validation-%s.grafana.com' % $._config.namespace)
      + validatingWebhook.withAdmissionReviewVersions(['v1'])
      + validatingWebhook.withFailurePolicy(if $._config.ignore_rollout_operator_zpdb_validation_webhook_failures then 'Ignore' else 'Fail')
      + validatingWebhook.withSideEffects('None')
      + validatingWebhook.withRulesMixin([
        {
          apiGroups: ['rollout-operator.grafana.com'],
          apiVersions: ['v1'],
          operations: ['CREATE', 'UPDATE'],
          resources: ['zoneawarepoddisruptionbudgets'],
          scope: 'Namespaced',
        },
      ])
      + {
        namespaceSelector: {
          matchLabels: {
            'kubernetes.io/metadata.name': $._config.namespace,
          },
        },
        clientConfig: {
          service: {
            name: 'rollout-operator',
            namespace: $._config.namespace,
            path: '/admission/zpdb-validation',
            port: 443,
          },
        },
      },
    ]),

  pod_eviction_webhook: if !enableWebhooks then null else
    validatingWebhookConfiguration.new('pod-eviction-%s' % $._config.namespace) +
    validatingWebhookConfiguration.mixin.metadata.withLabels({
      'grafana.com/namespace': $._config.namespace,
      'grafana.com/inject-rollout-operator-ca': 'true',
    }) +
    validatingWebhookConfiguration.withWebhooksMixin([
      validatingWebhook.withName('pod-eviction-%s.grafana.com' % $._config.namespace)
      + validatingWebhook.withAdmissionReviewVersions(['v1'])
      + validatingWebhook.withFailurePolicy(if $._config.ignore_rollout_operator_zpdb_eviction_webhook_failures then 'Ignore' else 'Fail')
      + validatingWebhook.withSideEffects('None')
      + validatingWebhook.withRulesMixin([
        {
          apiGroups: [''],
          apiVersions: ['v1'],
          operations: ['CREATE'],
          resources: ['pods/eviction'],
          scope: 'Namespaced',
        },
      ])
      + {
        namespaceSelector: {
          matchLabels: {
            'kubernetes.io/metadata.name': $._config.namespace,
          },
        },
        clientConfig: {
          service: {
            name: 'rollout-operator',
            namespace: $._config.namespace,
            path: '/admission/pod-eviction',
            port: 443,
          },
        },
      },
    ]),

  no_downscale_webhook: if !enableWebhooks then null else
    validatingWebhookConfiguration.new('no-downscale-%s' % $._config.namespace) +
    validatingWebhookConfiguration.mixin.metadata.withLabels({
      'grafana.com/namespace': $._config.namespace,
      'grafana.com/inject-rollout-operator-ca': 'true',
    }) +
    validatingWebhookConfiguration.withWebhooksMixin([
      validatingWebhook.withName('no-downscale-%s.grafana.com' % $._config.namespace)
      + validatingWebhook.withAdmissionReviewVersions(['v1'])
      + validatingWebhook.withFailurePolicy(if $._config.ignore_rollout_operator_no_downscale_webhook_failures then 'Ignore' else 'Fail')
      + validatingWebhook.withMatchPolicy('Equivalent')
      + validatingWebhook.withSideEffects('None')
      + validatingWebhook.withTimeoutSeconds(10)
      + validatingWebhook.withRulesMixin([
        {
          apiGroups: ['apps'],
          apiVersions: ['v1'],
          operations: ['UPDATE'],
          resources: ['statefulsets', 'statefulsets/scale'],
          scope: 'Namespaced',
        },
      ])
      + {
        namespaceSelector: {
          matchLabels: {
            'kubernetes.io/metadata.name': $._config.namespace,
          },
        },
        clientConfig: {
          service: {
            name: 'rollout-operator',
            namespace: $._config.namespace,
            path: '/admission/no-downscale',
            port: 443,
          },
        },
      },
    ]),

  prepare_downscale_webhook: if !enableWebhooks then null else
    mutatingWebhookConfiguration.new('prepare-downscale-%s' % $._config.namespace) +
    mutatingWebhookConfiguration.mixin.metadata.withLabels({
      'grafana.com/namespace': $._config.namespace,
      'grafana.com/inject-rollout-operator-ca': 'true',
    }) +
    mutatingWebhookConfiguration.withWebhooksMixin([
      mutatingWebhook.withName('prepare-downscale-%s.grafana.com' % $._config.namespace)
      + mutatingWebhook.withAdmissionReviewVersions(['v1'])
      + mutatingWebhook.withFailurePolicy(if $._config.ignore_rollout_operator_prepare_downscale_webhook_failures then 'Ignore' else 'Fail')
      + mutatingWebhook.withMatchPolicy('Equivalent')
      + mutatingWebhook.withSideEffects('NoneOnDryRun')
      + mutatingWebhook.withTimeoutSeconds(10)
      + mutatingWebhook.withRulesMixin([
        {
          apiGroups: ['apps'],
          apiVersions: ['v1'],
          operations: ['UPDATE'],
          resources: ['statefulsets', 'statefulsets/scale'],
          scope: 'Namespaced',
        },
      ])
      + {
        namespaceSelector: {
          matchLabels: {
            'kubernetes.io/metadata.name': $._config.namespace,
          },
        },
        clientConfig: {
          service: {
            name: 'rollout-operator',
            namespace: $._config.namespace,
            path: '/admission/prepare-downscale',
            port: 443,
          },
        },
      },
    ]),

  rollout_operator_service_account: if !$._config.rollout_operator_enabled then null else
    serviceAccount.new('rollout-operator') + serviceAccount.mixin.metadata.withNamespace($._config.namespace),
}
