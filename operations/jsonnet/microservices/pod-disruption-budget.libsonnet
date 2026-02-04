{
  local k = import 'k.libsonnet',

  local pdb = k.policy.v1.podDisruptionBudget,

  _config+:: {

    backend_scheduler+: {
      pdb: {
        enabled: false,
        max_unavailable: 1,
      },
    },

    backend_worker+: {
      pdb: {
        enabled: false,
        max_unavailable: 1,
      },
    },

    block_builder+: {
      pdb: {
        enabled: false,
        max_unavailable: 1,
      },
    },

    distributor+: {
      pdb: {
        enabled: false,
        max_unavailable: 1,
      },
    },

    live_store+: {
      // Live-stores use the zone-aware Pod Disruption Budget.
      // https://github.com/grafana/rollout-operator/blob/main/operations/rollout-operator/zone-aware-pod-disruption-budget.libsonnet
      zpdb: {
        enabled: true,  // Required by default.
        max_unavailable: 1,
        partition_regex: '[a-z\\-]+-zone-[a-z]-([0-9]+)',
        partition_group: 1,
      },
    },

    metrics_generator+: {
      pdb: {
        enabled: false,
        max_unavailable: 1,
      },
    },

    query_frontend+: {
      pdb: {
        enabled: false,
        max_unavailable: 1,
      },
    },

    querier+: {
      pdb: {
        enabled: false,
        max_unavailable: 1,
      },
    },

    memcached+: {
      pdb: {
        enabled: false,
        max_unavailable: 1,
      },
    },

  },

  pdbForController(controller, configKey)::
    assert std.objectHas($._config, configKey) : '$._config must have key ' + configKey;

    assert std.objectHas($._config[configKey], 'pdb') : '$._config.%s must have key pdb' % configKey;

    local pdbConfig = $._config[configKey].pdb;

    local maxUnavailable =
      if std.objectHas(pdbConfig, 'max_unavailable') && pdbConfig.max_unavailable != null then
        pdbConfig.max_unavailable
      else
        1;

    if !pdbConfig.enabled then
      {}
    else
      pdb.new(controller.metadata.name)
      + pdb.metadata.withLabels({ name: controller.metadata.name })
      + pdb.spec.withMaxUnavailable(maxUnavailable)
      + pdb.spec.selector.withMatchLabels({ name: controller.metadata.name }),
}
