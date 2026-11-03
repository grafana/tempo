{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='resourceClaimConsumerReference', url='', help='"ResourceClaimConsumerReference contains enough information to let you locate the consumer of a ResourceClaim. The user must be a resource in the same namespace as the ResourceClaim."'),
  '#withApiGroup':: d.fn(help='"APIGroup is the group for the resource being referenced. It is empty for the core API. This matches the group in the APIVersion that is used when creating the resources."', args=[d.arg(name='apiGroup', type=d.T.string)]),
  withApiGroup(apiGroup): { apiGroup: apiGroup },
  '#withName':: d.fn(help='"Name is the name of resource being referenced."', args=[d.arg(name='name', type=d.T.string)]),
  withName(name): { name: name },
  '#withResource':: d.fn(help='"Resource is the type of resource being referenced, for example \\"pods\\"."', args=[d.arg(name='resource', type=d.T.string)]),
  withResource(resource): { resource: resource },
  '#withUid':: d.fn(help='"UID identifies exactly one incarnation of the resource."', args=[d.arg(name='uid', type=d.T.string)]),
  withUid(uid): { uid: uid },
  '#mixin': 'ignore',
  mixin: self,
}
