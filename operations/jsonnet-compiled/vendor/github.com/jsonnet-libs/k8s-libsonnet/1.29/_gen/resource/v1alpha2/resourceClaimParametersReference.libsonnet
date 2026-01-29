{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='resourceClaimParametersReference', url='', help='"ResourceClaimParametersReference contains enough information to let you locate the parameters for a ResourceClaim. The object must be in the same namespace as the ResourceClaim."'),
  '#withApiGroup':: d.fn(help='"APIGroup is the group for the resource being referenced. It is empty for the core API. This matches the group in the APIVersion that is used when creating the resources."', args=[d.arg(name='apiGroup', type=d.T.string)]),
  withApiGroup(apiGroup): { apiGroup: apiGroup },
  '#withKind':: d.fn(help="\"Kind is the type of resource being referenced. This is the same value as in the parameter object's metadata, for example \\\"ConfigMap\\\".\"", args=[d.arg(name='kind', type=d.T.string)]),
  withKind(kind): { kind: kind },
  '#withName':: d.fn(help='"Name is the name of resource being referenced."', args=[d.arg(name='name', type=d.T.string)]),
  withName(name): { name: name },
  '#mixin': 'ignore',
  mixin: self,
}
