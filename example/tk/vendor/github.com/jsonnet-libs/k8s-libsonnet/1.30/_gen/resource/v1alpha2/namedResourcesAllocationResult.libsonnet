{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='namedResourcesAllocationResult', url='', help='"NamedResourcesAllocationResult is used in AllocationResultModel."'),
  '#withName':: d.fn(help='"Name is the name of the selected resource instance."', args=[d.arg(name='name', type=d.T.string)]),
  withName(name): { name: name },
  '#mixin': 'ignore',
  mixin: self,
}
