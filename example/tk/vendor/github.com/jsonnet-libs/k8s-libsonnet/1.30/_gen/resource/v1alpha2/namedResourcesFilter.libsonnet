{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='namedResourcesFilter', url='', help='"NamedResourcesFilter is used in ResourceFilterModel."'),
  '#withSelector':: d.fn(help='"Selector is a CEL expression which must evaluate to true if a resource instance is suitable. The language is as defined in https://kubernetes.io/docs/reference/using-api/cel/\\n\\nIn addition, for each type NamedResourcesin AttributeValue there is a map that resolves to the corresponding value of the instance under evaluation. For example:\\n\\n   attributes.quantity[\\"a\\"].isGreaterThan(quantity(\\"0\\")) &&\\n   attributes.stringslice[\\"b\\"].isSorted()"', args=[d.arg(name='selector', type=d.T.string)]),
  withSelector(selector): { selector: selector },
  '#mixin': 'ignore',
  mixin: self,
}
