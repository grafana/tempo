{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='forNode', url='', help='"ForNode provides information about which nodes should consume this endpoint."'),
  '#withName':: d.fn(help='"name represents the name of the node."', args=[d.arg(name='name', type=d.T.string)]),
  withName(name): { name: name },
  '#mixin': 'ignore',
  mixin: self,
}
