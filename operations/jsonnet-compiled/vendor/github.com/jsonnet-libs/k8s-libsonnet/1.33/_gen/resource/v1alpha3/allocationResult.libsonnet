{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='allocationResult', url='', help='"AllocationResult contains attributes of an allocated resource."'),
  '#devices':: d.obj(help='"DeviceAllocationResult is the result of allocating devices."'),
  devices: {
    '#withConfig':: d.fn(help='"This field is a combination of all the claim and class configuration parameters. Drivers can distinguish between those based on a flag.\\n\\nThis includes configuration parameters for drivers which have no allocated devices in the result because it is up to the drivers which configuration parameters they support. They can silently ignore unknown configuration parameters."', args=[d.arg(name='config', type=d.T.array)]),
    withConfig(config): { devices+: { config: if std.isArray(v=config) then config else [config] } },
    '#withConfigMixin':: d.fn(help='"This field is a combination of all the claim and class configuration parameters. Drivers can distinguish between those based on a flag.\\n\\nThis includes configuration parameters for drivers which have no allocated devices in the result because it is up to the drivers which configuration parameters they support. They can silently ignore unknown configuration parameters."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='config', type=d.T.array)]),
    withConfigMixin(config): { devices+: { config+: if std.isArray(v=config) then config else [config] } },
    '#withResults':: d.fn(help='"Results lists all allocated devices."', args=[d.arg(name='results', type=d.T.array)]),
    withResults(results): { devices+: { results: if std.isArray(v=results) then results else [results] } },
    '#withResultsMixin':: d.fn(help='"Results lists all allocated devices."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='results', type=d.T.array)]),
    withResultsMixin(results): { devices+: { results+: if std.isArray(v=results) then results else [results] } },
  },
  '#nodeSelector':: d.obj(help='"A node selector represents the union of the results of one or more label queries over a set of nodes; that is, it represents the OR of the selectors represented by the node selector terms."'),
  nodeSelector: {
    '#withNodeSelectorTerms':: d.fn(help='"Required. A list of node selector terms. The terms are ORed."', args=[d.arg(name='nodeSelectorTerms', type=d.T.array)]),
    withNodeSelectorTerms(nodeSelectorTerms): { nodeSelector+: { nodeSelectorTerms: if std.isArray(v=nodeSelectorTerms) then nodeSelectorTerms else [nodeSelectorTerms] } },
    '#withNodeSelectorTermsMixin':: d.fn(help='"Required. A list of node selector terms. The terms are ORed."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='nodeSelectorTerms', type=d.T.array)]),
    withNodeSelectorTermsMixin(nodeSelectorTerms): { nodeSelector+: { nodeSelectorTerms+: if std.isArray(v=nodeSelectorTerms) then nodeSelectorTerms else [nodeSelectorTerms] } },
  },
  '#mixin': 'ignore',
  mixin: self,
}
