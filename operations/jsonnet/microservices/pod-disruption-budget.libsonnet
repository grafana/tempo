{
  local k = import 'k.libsonnet',

  local pdb = k.policy.v1.podDisruptionBudget,

  _config+:: {

    backend_scheduler+: {
      pdb: {
        enabled: false,
        maxUnavailable: 1,
      },
    },

    backend_worker+: {
      pdb: {
        enabled: false,
        maxUnavailable: 1,
      },
    },

    block_builder+: {
      pdb: {
        enabled: false,
        maxUnavailable: 1,
      },
    },

    distributor+: {
      pdb: {
        enabled: false,
        maxUnavailable: 1,
      },
    },

    // TODO: use the zpdb from the rollout-operator for live_stores
    // live_store+: {
    //   pdb: {
    //     enabled: false,
    //   },
    // },

    metrics_generator+: {
      pdb: {
        enabled: false,
        maxUnavailable: 1,
      },
    },

    query_frontend+: {
      pdb: {
        enabled: false,
        maxUnavailable: 1,
      },
    },

    querier+: {
      pdb: {
        enabled: false,
        maxUnavailable: 1,
      },
    },

    memcached+: {
      pdb: {
        enabled: false,
        maxUnavailable: 1,
      },
    },

  },

  pdbForController(controller, configKey)::
    assert std.objectHas($._config, configKey) : '$._config must have key ' + configKey;

    assert std.objectHas($._config[configKey], 'pdb') : '$._config.%s must have key pdb' % configKey;

    local pdbConfig = $._config[configKey].pdb;

    local maxUnavailable =
      if std.objectHas(pdbConfig, 'maxUnavailable') && pdbConfig.maxUnavailable != null then
        pdbConfig.maxUnavailable
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
