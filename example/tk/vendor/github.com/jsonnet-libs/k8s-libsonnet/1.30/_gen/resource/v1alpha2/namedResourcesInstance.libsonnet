{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='namedResourcesInstance', url='', help='"NamedResourcesInstance represents one individual hardware instance that can be selected based on its attributes."'),
  '#withAttributes':: d.fn(help='"Attributes defines the attributes of this resource instance. The name of each attribute must be unique."', args=[d.arg(name='attributes', type=d.T.array)]),
  withAttributes(attributes): { attributes: if std.isArray(v=attributes) then attributes else [attributes] },
  '#withAttributesMixin':: d.fn(help='"Attributes defines the attributes of this resource instance. The name of each attribute must be unique."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='attributes', type=d.T.array)]),
  withAttributesMixin(attributes): { attributes+: if std.isArray(v=attributes) then attributes else [attributes] },
  '#withName':: d.fn(help='"Name is unique identifier among all resource instances managed by the driver on the node. It must be a DNS subdomain."', args=[d.arg(name='name', type=d.T.string)]),
  withName(name): { name: name },
  '#mixin': 'ignore',
  mixin: self,
}
