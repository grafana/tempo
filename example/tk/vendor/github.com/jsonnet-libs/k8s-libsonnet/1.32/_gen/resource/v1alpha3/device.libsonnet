{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='device', url='', help='"Device represents one individual hardware instance that can be selected based on its attributes. Besides the name, exactly one field must be set."'),
  '#basic':: d.obj(help='"BasicDevice defines one device instance."'),
  basic: {
    '#withAttributes':: d.fn(help='"Attributes defines the set of attributes for this device. The name of each attribute must be unique in that set.\\n\\nThe maximum number of attributes and capacities combined is 32."', args=[d.arg(name='attributes', type=d.T.object)]),
    withAttributes(attributes): { basic+: { attributes: attributes } },
    '#withAttributesMixin':: d.fn(help='"Attributes defines the set of attributes for this device. The name of each attribute must be unique in that set.\\n\\nThe maximum number of attributes and capacities combined is 32."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='attributes', type=d.T.object)]),
    withAttributesMixin(attributes): { basic+: { attributes+: attributes } },
    '#withCapacity':: d.fn(help='"Capacity defines the set of capacities for this device. The name of each capacity must be unique in that set.\\n\\nThe maximum number of attributes and capacities combined is 32."', args=[d.arg(name='capacity', type=d.T.object)]),
    withCapacity(capacity): { basic+: { capacity: capacity } },
    '#withCapacityMixin':: d.fn(help='"Capacity defines the set of capacities for this device. The name of each capacity must be unique in that set.\\n\\nThe maximum number of attributes and capacities combined is 32."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='capacity', type=d.T.object)]),
    withCapacityMixin(capacity): { basic+: { capacity+: capacity } },
  },
  '#withName':: d.fn(help='"Name is unique identifier among all devices managed by the driver in the pool. It must be a DNS label."', args=[d.arg(name='name', type=d.T.string)]),
  withName(name): { name: name },
  '#mixin': 'ignore',
  mixin: self,
}
