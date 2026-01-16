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

  },

  local vpa = import 'vpa.libsonnet',
  local verticalPodAutoscaler = vpa.autoscaling.v1.verticalPodAutoscaler,
  vpaForController(controller, configKey)::
    assert controller.kind == 'Deployment'
           || controller.kind == 'StatefulSet'
           || controller.kind == 'DaemonSet'
           : 'must provide known controller resource';

    assert std.objectHas($._config, configKey) : '$._config must have key ' + configKey;

    local vpaConfig = $._config[configKey].vpa;

    if !vpaConfig.enabled then
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
        + verticalPodAutoscaler.spec.resourcePolicy.containerPolicies.withMaxAllowed({
          cpu: vpaConfig.cpu.max,
          memory: vpaConfig.memory.max,
        })
        + verticalPodAutoscaler.spec.resourcePolicy.containerPolicies.withMinAllowed({
          cpu: vpaConfig.cpu.min,
          memory: vpaConfig.memory.min,
        }),
      ]),

}
