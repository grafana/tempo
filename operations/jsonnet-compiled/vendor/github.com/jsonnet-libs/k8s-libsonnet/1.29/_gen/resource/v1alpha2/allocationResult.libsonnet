{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='allocationResult', url='', help='"AllocationResult contains attributes of an allocated resource."'),
  '#availableOnNodes':: d.obj(help='"A node selector represents the union of the results of one or more label queries over a set of nodes; that is, it represents the OR of the selectors represented by the node selector terms."'),
  availableOnNodes: {
    '#withNodeSelectorTerms':: d.fn(help='"Required. A list of node selector terms. The terms are ORed."', args=[d.arg(name='nodeSelectorTerms', type=d.T.array)]),
    withNodeSelectorTerms(nodeSelectorTerms): { availableOnNodes+: { nodeSelectorTerms: if std.isArray(v=nodeSelectorTerms) then nodeSelectorTerms else [nodeSelectorTerms] } },
    '#withNodeSelectorTermsMixin':: d.fn(help='"Required. A list of node selector terms. The terms are ORed."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='nodeSelectorTerms', type=d.T.array)]),
    withNodeSelectorTermsMixin(nodeSelectorTerms): { availableOnNodes+: { nodeSelectorTerms+: if std.isArray(v=nodeSelectorTerms) then nodeSelectorTerms else [nodeSelectorTerms] } },
  },
  '#withResourceHandles':: d.fn(help='"ResourceHandles contain the state associated with an allocation that should be maintained throughout the lifetime of a claim. Each ResourceHandle contains data that should be passed to a specific kubelet plugin once it lands on a node. This data is returned by the driver after a successful allocation and is opaque to Kubernetes. Driver documentation may explain to users how to interpret this data if needed.\\n\\nSetting this field is optional. It has a maximum size of 32 entries. If null (or empty), it is assumed this allocation will be processed by a single kubelet plugin with no ResourceHandle data attached. The name of the kubelet plugin invoked will match the DriverName set in the ResourceClaimStatus this AllocationResult is embedded in."', args=[d.arg(name='resourceHandles', type=d.T.array)]),
  withResourceHandles(resourceHandles): { resourceHandles: if std.isArray(v=resourceHandles) then resourceHandles else [resourceHandles] },
  '#withResourceHandlesMixin':: d.fn(help='"ResourceHandles contain the state associated with an allocation that should be maintained throughout the lifetime of a claim. Each ResourceHandle contains data that should be passed to a specific kubelet plugin once it lands on a node. This data is returned by the driver after a successful allocation and is opaque to Kubernetes. Driver documentation may explain to users how to interpret this data if needed.\\n\\nSetting this field is optional. It has a maximum size of 32 entries. If null (or empty), it is assumed this allocation will be processed by a single kubelet plugin with no ResourceHandle data attached. The name of the kubelet plugin invoked will match the DriverName set in the ResourceClaimStatus this AllocationResult is embedded in."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='resourceHandles', type=d.T.array)]),
  withResourceHandlesMixin(resourceHandles): { resourceHandles+: if std.isArray(v=resourceHandles) then resourceHandles else [resourceHandles] },
  '#withShareable':: d.fn(help='"Shareable determines whether the resource supports more than one consumer at a time."', args=[d.arg(name='shareable', type=d.T.boolean)]),
  withShareable(shareable): { shareable: shareable },
  '#mixin': 'ignore',
  mixin: self,
}
