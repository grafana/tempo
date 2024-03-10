{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='resourceClaimSchedulingStatus', url='', help='"ResourceClaimSchedulingStatus contains information about one particular ResourceClaim with \\"WaitForFirstConsumer\\" allocation mode."'),
  '#withName':: d.fn(help='"Name matches the pod.spec.resourceClaims[*].Name field."', args=[d.arg(name='name', type=d.T.string)]),
  withName(name): { name: name },
  '#withUnsuitableNodes':: d.fn(help='"UnsuitableNodes lists nodes that the ResourceClaim cannot be allocated for.\\n\\nThe size of this field is limited to 128, the same as for PodSchedulingSpec.PotentialNodes. This may get increased in the future, but not reduced."', args=[d.arg(name='unsuitableNodes', type=d.T.array)]),
  withUnsuitableNodes(unsuitableNodes): { unsuitableNodes: if std.isArray(v=unsuitableNodes) then unsuitableNodes else [unsuitableNodes] },
  '#withUnsuitableNodesMixin':: d.fn(help='"UnsuitableNodes lists nodes that the ResourceClaim cannot be allocated for.\\n\\nThe size of this field is limited to 128, the same as for PodSchedulingSpec.PotentialNodes. This may get increased in the future, but not reduced."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='unsuitableNodes', type=d.T.array)]),
  withUnsuitableNodesMixin(unsuitableNodes): { unsuitableNodes+: if std.isArray(v=unsuitableNodes) then unsuitableNodes else [unsuitableNodes] },
  '#mixin': 'ignore',
  mixin: self,
}
