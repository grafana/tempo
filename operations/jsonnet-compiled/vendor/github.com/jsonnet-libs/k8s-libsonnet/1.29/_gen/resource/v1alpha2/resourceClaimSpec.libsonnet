{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='resourceClaimSpec', url='', help='"ResourceClaimSpec defines how a resource is to be allocated."'),
  '#parametersRef':: d.obj(help='"ResourceClaimParametersReference contains enough information to let you locate the parameters for a ResourceClaim. The object must be in the same namespace as the ResourceClaim."'),
  parametersRef: {
    '#withApiGroup':: d.fn(help='"APIGroup is the group for the resource being referenced. It is empty for the core API. This matches the group in the APIVersion that is used when creating the resources."', args=[d.arg(name='apiGroup', type=d.T.string)]),
    withApiGroup(apiGroup): { parametersRef+: { apiGroup: apiGroup } },
    '#withKind':: d.fn(help="\"Kind is the type of resource being referenced. This is the same value as in the parameter object's metadata, for example \\\"ConfigMap\\\".\"", args=[d.arg(name='kind', type=d.T.string)]),
    withKind(kind): { parametersRef+: { kind: kind } },
    '#withName':: d.fn(help='"Name is the name of resource being referenced."', args=[d.arg(name='name', type=d.T.string)]),
    withName(name): { parametersRef+: { name: name } },
  },
  '#withAllocationMode':: d.fn(help='"Allocation can start immediately or when a Pod wants to use the resource. \\"WaitForFirstConsumer\\" is the default."', args=[d.arg(name='allocationMode', type=d.T.string)]),
  withAllocationMode(allocationMode): { allocationMode: allocationMode },
  '#withResourceClassName':: d.fn(help='"ResourceClassName references the driver and additional parameters via the name of a ResourceClass that was created as part of the driver deployment."', args=[d.arg(name='resourceClassName', type=d.T.string)]),
  withResourceClassName(resourceClassName): { resourceClassName: resourceClassName },
  '#mixin': 'ignore',
  mixin: self,
}
