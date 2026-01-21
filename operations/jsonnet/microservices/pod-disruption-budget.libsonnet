{
  local k = import 'k.libsonnet',

  local pdb = k.policy.v1.podDisruptionBudget,

  _config+:: {

    backend_scheduler+: {
      pdb: {
        enabled: false,
      },
    },

    backend_worker+: {
      pdb: {
        enabled: false,
      },
    },

    block_builder+: {
      pdb: {
        enabled: false,
      },
    },

    distributor+: {
      pdb: {
        enabled: false,
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
      },
    },

    query_frontend+: {
      pdb: {
        enabled: false,
      },
    },

    querier+: {
      pdb: {
        enabled: false,
      },
    },

    memcached+: {
      pdb: {
        enabled: false,
      },
    },

  },


  pdbForController(controller, configKey)::
    assert std.objectHas($._config, configKey) : '$._config must have key ' + configKey;

    assert std.objectHas($._config[configKey], 'pdb') : '$._config.%s must have key pdb' % configKey;

    local pdbConfig = $._config[configKey].pdb;

    if !pdbConfig.enabled then
      {}
    else
      pdb.new(controller.metadata.new)
      + pdb.spec.withMaxUnavailable(1)
      + pdb.spec.selector.withMatchLabels({ name: controller.metadata.name }),
}
