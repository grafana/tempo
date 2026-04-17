local d = import 'doc-util/main.libsonnet';

local withTargetRef = {
  '#withTargetRef':: d.fn(help='Set spec.TargetRef to `object`', args=[d.arg(name='object', type=d.T.object)]),
  withTargetRef(object):
    { spec+: { targetRef+: {
      apiVersion: object.apiVersion,
      kind: object.kind,
      name: object.metadata.name,
    } } },
};

local patch = {
  verticalPodAutoscaler+: {
    spec+: withTargetRef,
  },

  verticalPodAutoscalerContainerResourcePolicy+: {
    '#':: d.pkg(
      name='verticalPodAutoscalerContainerResourcePolicy',
      url='',
      help='"An array of these is used as the input to `verticalPodAutoscaler.spec.resourcePolicy.withContainerPolicies()`."',
    ),

    '#withContainerName': d.fn(
        'The name of the container that the policy applies to. If not specified, the policy serves as the default policy.',
        [d.arg('name', d.T.string)]
    ),
    withContainerName(name):: {
        containerName: name,
    },

    '#withControlledResources': d.fn(
        'Specifies the type of recommendations that will be computed (and possibly applied) by VPA. If not specified, the default of [ResourceCPU, ResourceMemory] will be used.',
        [d.arg('resources', d.T.array)]
    ),
    withControlledResources(resources):: {
        controlledResources: std.uniq(std.sort(resources)),
    },

    '#withControlledResourcesMixin': d.fn(
        'withControlledResourcesMixin is like withControlledResources, but appends to the existing list',
        [d.arg('resources', d.T.array)]
    ),
    withControlledResourcesMixin(resources):: {
        controlledResources: std.uniq(std.sort(super.resources+resources)),
    },

    '#withControlledValues': d.fn(
        'Which resource values should be controlled by VPA. Valid values are "RequestsAndLimits" and "RequestsOnly". The default is "RequestsAndLimits".',
        [d.arg('values', d.T.string)]
    ),
    withControlledValues(values):: {
        controlledValues: values,
    },

    '#withMaxAllowed': d.fn(
      'Specifies the maximum amount of resources that will be recommended for the container. The default is no maximum.',
      [d.arg('maxAllowed', d.T.object)],
    ),
    withMaxAllowed(maxAllowed):: {
      maxAllowed: maxAllowed,
    },

    '#withMaxAllowedMixin': d.fn(
      'Like withMaxAllowed but merges with the existing object.',
      [d.arg('maxAllowed', d.T.object)],
    ),
    withMaxAllowedMixin(maxAllowed):: {
      maxAllowed+: maxAllowed,
    },

    '#withMinAllowed': d.fn(
      'Specifies the minimal amount of resources that will be recommended for the container. The default is no minimum.',
      [d.arg('maxAllowed', d.T.object)],
    ),
    withMinAllowed(minAllowed):: {
      minAllowed: minAllowed,
    },

    '#withMinAllowedMixin': d.fn(
      'Like withMinAllowed but merges with the existing object.',
      [d.arg('minAllowed', d.T.object)],
    ),
    withMinAllowedMixin(minAllowed):: {
      minAllowed+: minAllowed,
    },

    '#withMode': d.fn(
      'Whether autoscaler is enabled for the container. Valid values are "Off" and "Auto". The default is "Auto".',
      [d.arg('minAllowed', d.T.string)],
    ),
    withMode(mode):: {
      mode: mode,
    },
  },
};

{
  autoscaling+:: {
    v1+: patch,
    v1beta2+: patch,
  },
}
