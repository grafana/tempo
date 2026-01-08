{
  // newZPDB creates new ZoneAwarePodDisruptionBudget resource.
  // name - a unique name within the namespace for this resource. eg ingester-rollout
  // rolloutGroup - the rollout-group label to match pods who are within this zpdb. eg ingester
  // maxUnavailable - the max unavailable pod count. Can be expressed as an integer >= 0 or as a percentage 50% (0-100%)
  // podNamePartitionRegex - use to enable partition awareness. A regex which can extract a partition name from a pod name. ie [a-z\\-]+-zone-[a-z]-([0-9]+)
  // podNameRegexGroup - use to define the regex grouping which holds the partition name. default is 1
  newZPDB(name, rolloutGroup, maxUnavailable, podNamePartitionRegex='', podNameRegexGroup=1):: {
    assert $._config.rollout_operator_webhooks_enabled : 'zpdb configuration requires rollout_operator_webhooks_enabled=true',
    apiVersion: $.zpdb_template.spec.group + '/v1',
    kind: $.zpdb_template.spec.names.kind,
    metadata: {
      name: name,
      namespace: $._config.namespace,
      labels: {
        name: name,
      },
    },
    spec: (
      {
        selector: {
          matchLabels: {
            'rollout-group': rolloutGroup,
          },
        },
      }
      + (if std.isString(maxUnavailable) && std.endsWith(maxUnavailable, '%') then {
           maxUnavailablePercentage: std.parseInt(std.stripChars(maxUnavailable, '%')),
         } else if std.isString(maxUnavailable) then {
           maxUnavailable: std.parseInt(maxUnavailable),
         } else {
           maxUnavailable: maxUnavailable,
         })
      + (if podNamePartitionRegex == '' then {} else {
           podNamePartitionRegex: podNamePartitionRegex,
           podNameRegexGroup: podNameRegexGroup,
         })
    ),
  },
}
