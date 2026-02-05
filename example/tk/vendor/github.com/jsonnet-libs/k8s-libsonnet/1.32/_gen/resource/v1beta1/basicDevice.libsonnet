{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='basicDevice', url='', help='"BasicDevice defines one device instance."'),
  '#withAttributes':: d.fn(help='"Attributes defines the set of attributes for this device. The name of each attribute must be unique in that set.\\n\\nThe maximum number of attributes and capacities combined is 32."', args=[d.arg(name='attributes', type=d.T.object)]),
  withAttributes(attributes): { attributes: attributes },
  '#withAttributesMixin':: d.fn(help='"Attributes defines the set of attributes for this device. The name of each attribute must be unique in that set.\\n\\nThe maximum number of attributes and capacities combined is 32."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='attributes', type=d.T.object)]),
  withAttributesMixin(attributes): { attributes+: attributes },
  '#withCapacity':: d.fn(help='"Capacity defines the set of capacities for this device. The name of each capacity must be unique in that set.\\n\\nThe maximum number of attributes and capacities combined is 32."', args=[d.arg(name='capacity', type=d.T.object)]),
  withCapacity(capacity): { capacity: capacity },
  '#withCapacityMixin':: d.fn(help='"Capacity defines the set of capacities for this device. The name of each capacity must be unique in that set.\\n\\nThe maximum number of attributes and capacities combined is 32."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='capacity', type=d.T.object)]),
  withCapacityMixin(capacity): { capacity+: capacity },
  '#mixin': 'ignore',
  mixin: self,
}
