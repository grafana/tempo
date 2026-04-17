{
  // replicaTemplate creates new ReplicaTemplate resource.
  // If replicas is > 0, spec.replicas field is specified in the resource, if replicas <= 0, spec.replicas field is hidden.
  // Syntactically valid label selector is required, and may be used by HorizontalPodAutoscaler controller when ReplicaTemplate
  // is used as scaled resource depending on metric target type.
  // (When using targetType=AverageValue, label selector is not used for scaling computation).
  replicaTemplate(name, replicas, label_selector):: {
    assert $._config.rollout_operator_webhooks_enabled : 'replica template configuration requires rollout_operator_webhooks_enabled=true',
    apiVersion: $.replica_template.spec.group + '/v1',
    kind: $.replica_template.spec.names.kind,
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
