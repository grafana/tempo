{
  _config+:: {

    backend_scheduler+: {
      vpa: {
        enabled: false,
        update_mode: 'Auto',
        target_resources: ['cpu', 'memory'],

        // TODO: are the CPU/Memory min/max values the same as the
        // requests/limits in the config?  Can we just reuse those values
        // instead of duplicating them here?

        cpu: {
          min: '100m',
          max: '2',
        },
        memory: {
          min: '1Gi',
          max: '4Gi',
        },
      },
    },

    backend_worker+: {
      vpa: {
        enabled: false,
        update_mode: 'Auto',
        target_resources: ['cpu', 'memory'],

        cpu: {
          min: '100m',
          max: '2',
        },
        memory: {
          min: '1Gi',
          max: '4Gi',
        },
      },
    },

    block_builder+: {
      vpa: {
        enabled: false,
        update_mode: 'Auto',
        target_resources: ['cpu', 'memory'],

        cpu: {
          min: '100m',
          max: '2',
        },
        memory: {
          min: '1Gi',
          max: '4Gi',
        },
      },
    },

    distributor+: {
      vpa: {
        enabled: false,
        update_mode: 'Auto',
        target_resources: ['cpu', 'memory'],

        cpu: {
          min: '100m',
          max: '2',
        },
        memory: {
          min: '1Gi',
          max: '4Gi',
        },
      },
    },

    live_store+: {
      vpa: {
        enabled: false,
        update_mode: 'Auto',
        target_resources: ['cpu', 'memory'],

        cpu: {
          min: '100m',
          max: '2',
        },
        memory: {
          min: '1Gi',
          max: '4Gi',
        },
      },
    },

    metrics_generator+: {
      vpa: {
        enabled: false,
        update_mode: 'Auto',
        target_resources: ['cpu', 'memory'],

        cpu: {
          min: '100m',
          max: '2',
        },
        memory: {
          min: '1Gi',
          max: '4Gi',
        },
      },
    },

    querier+: {
      vpa: {
        enabled: false,
        update_mode: 'Auto',
        target_resources: ['cpu', 'memory'],

        cpu: {
          min: '100m',
          max: '2',
        },
        memory: {
          min: '1Gi',
          max: '4Gi',
        },
      },
    },

    memcached+: {
      vpa: {
        enabled: false,
        update_mode: 'Auto',
        target_resources: ['cpu'],

        cpu: {
          min: '10m',
          max: '2',
        },
      },
    },

  },

  local vpa = import 'github.com/jsonnet-libs/vertical-pod-autoscaler-libsonnet/1.0.0/main.libsonnet',

  local verticalPodAutoscaler = vpa.autoscaling.v1.verticalPodAutoscaler,
  vpaForController(controller, configKey)::
    assert controller.kind == 'Deployment'
           || controller.kind == 'StatefulSet'
           || controller.kind == 'DaemonSet'
           : 'must provide known controller resource';

    assert std.objectHas($._config, configKey) : '$._config must have key ' + configKey;

    assert std.objectHas($._config[configKey], 'vpa') : '$._config.%s must have key "vpa"' % configKey;

    local vpaConfig = $._config[configKey].vpa;

    if std.objectHas(vpaConfig, 'enabled') && !vpaConfig.enabled then
      {}
    else
      verticalPodAutoscaler.new(controller.metadata.name)
      + verticalPodAutoscaler.spec.withTargetRef(controller)
      + verticalPodAutoscaler.spec.updatePolicy.withUpdateMode(vpaConfig.update_mode)
      + verticalPodAutoscaler.spec.updatePolicy.withMinReplicas(1)
      + verticalPodAutoscaler.spec.resourcePolicy.withContainerPolicies([
        verticalPodAutoscaler.spec.resourcePolicy.containerPolicies.withContainerName(controller.spec.template.spec.containers[0].name)
        + verticalPodAutoscaler.spec.resourcePolicy.containerPolicies.withMode('Auto')
        + verticalPodAutoscaler.spec.resourcePolicy.containerPolicies.withControlledValues('RequestsAndLimits')
        + verticalPodAutoscaler.spec.resourcePolicy.containerPolicies.withControlledResources(vpaConfig.target_resources)
        + (
          if std.objectHas(vpaConfig, 'cpu') && std.objectHas(vpaConfig.cpu, 'max') ||
             std.objectHas(vpaConfig, 'memory') && std.objectHas(vpaConfig.memory, 'max')
          then
            verticalPodAutoscaler.spec.resourcePolicy.containerPolicies.withMaxAllowed({
              cpu: if std.objectHas(vpaConfig, 'cpu') && std.objectHas(vpaConfig.cpu, 'max') then vpaConfig.cpu.max,
              memory: if std.objectHas(vpaConfig, 'memory') && std.objectHas(vpaConfig.memory, 'max') then vpaConfig.memory.max,
            })
        )
        + (
          if std.objectHas(vpaConfig, 'cpu') && std.objectHas(vpaConfig.cpu, 'min') ||
             std.objectHas(vpaConfig, 'memory') && std.objectHas(vpaConfig.memory, 'min')
          then
            verticalPodAutoscaler.spec.resourcePolicy.containerPolicies.withMinAllowed({
              cpu: if std.objectHas(vpaConfig, 'cpu') && std.objectHas(vpaConfig.cpu, 'min') then vpaConfig.cpu.min,
              memory: if std.objectHas(vpaConfig, 'memory') && std.objectHas(vpaConfig.memory, 'min') then vpaConfig.memory.min,
            })
        ),
      ]),

}
