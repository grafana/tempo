{
  _config+: {
    replica_template_custom_resource_definition_enabled: false,
  },

  replica_template:: std.parseYaml(importstr 'replica-templates.yaml'),
  replica_template_custom_resource: if !$._config.replica_template_custom_resource_definition_enabled then null else $.replica_template,

  // replicaTemplate creates new ReplicaTemplate resource.
  // If replicas is > 0, spec.replicas field is specified in the resource, if replicas <= 0, spec.replicas field is hidden.
  // Syntactically valid label selector is required, and may be used by HorizontalPodAutoscaler controller when ReplicaTemplate
  // is used as scaled resource depending on metric target type.
  // (When using targetType=AverageValue, label selector is not used for scaling computation).
  replicaTemplate(name, replicas, label_selector):: {
    apiVersion: 'rollout-operator.grafana.com/v1',
    kind: 'ReplicaTemplate',
    metadata: {
      name: name,
      namespace: $._config.namespace,
    },
    spec: {
      // HPA requires that label selector exists and is valid, but it will not be used for target type of AverageValue.
      labelSelector: label_selector,
    } + (
      if replicas <= 0 then {
        replicas:: null,  // Hide replicas field.
      } else {
        replicas: replicas,
      }
    ),
  },
}
