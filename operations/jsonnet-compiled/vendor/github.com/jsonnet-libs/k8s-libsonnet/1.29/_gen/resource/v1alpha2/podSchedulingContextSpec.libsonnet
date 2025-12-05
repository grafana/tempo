{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='podSchedulingContextSpec', url='', help='"PodSchedulingContextSpec describes where resources for the Pod are needed."'),
  '#withPotentialNodes':: d.fn(help='"PotentialNodes lists nodes where the Pod might be able to run.\\n\\nThe size of this field is limited to 128. This is large enough for many clusters. Larger clusters may need more attempts to find a node that suits all pending resources. This may get increased in the future, but not reduced."', args=[d.arg(name='potentialNodes', type=d.T.array)]),
  withPotentialNodes(potentialNodes): { potentialNodes: if std.isArray(v=potentialNodes) then potentialNodes else [potentialNodes] },
  '#withPotentialNodesMixin':: d.fn(help='"PotentialNodes lists nodes where the Pod might be able to run.\\n\\nThe size of this field is limited to 128. This is large enough for many clusters. Larger clusters may need more attempts to find a node that suits all pending resources. This may get increased in the future, but not reduced."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='potentialNodes', type=d.T.array)]),
  withPotentialNodesMixin(potentialNodes): { potentialNodes+: if std.isArray(v=potentialNodes) then potentialNodes else [potentialNodes] },
  '#withSelectedNode':: d.fn(help='"SelectedNode is the node for which allocation of ResourceClaims that are referenced by the Pod and that use \\"WaitForFirstConsumer\\" allocation is to be attempted."', args=[d.arg(name='selectedNode', type=d.T.string)]),
  withSelectedNode(selectedNode): { selectedNode: selectedNode },
  '#mixin': 'ignore',
  mixin: self,
}
