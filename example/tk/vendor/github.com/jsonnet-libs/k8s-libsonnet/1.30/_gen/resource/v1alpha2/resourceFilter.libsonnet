{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='resourceFilter', url='', help='"ResourceFilter is a filter for resources from one particular driver."'),
  '#namedResources':: d.obj(help='"NamedResourcesFilter is used in ResourceFilterModel."'),
  namedResources: {
    '#withSelector':: d.fn(help='"Selector is a CEL expression which must evaluate to true if a resource instance is suitable. The language is as defined in https://kubernetes.io/docs/reference/using-api/cel/\\n\\nIn addition, for each type NamedResourcesin AttributeValue there is a map that resolves to the corresponding value of the instance under evaluation. For example:\\n\\n   attributes.quantity[\\"a\\"].isGreaterThan(quantity(\\"0\\")) &&\\n   attributes.stringslice[\\"b\\"].isSorted()"', args=[d.arg(name='selector', type=d.T.string)]),
    withSelector(selector): { namedResources+: { selector: selector } },
  },
  '#withDriverName':: d.fn(help='"DriverName is the name used by the DRA driver kubelet plugin."', args=[d.arg(name='driverName', type=d.T.string)]),
  withDriverName(driverName): { driverName: driverName },
  '#mixin': 'ignore',
  mixin: self,
}
