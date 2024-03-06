{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='podSchedulingGate', url='', help='"PodSchedulingGate is associated to a Pod to guard its scheduling."'),
  '#withName':: d.fn(help='"Name of the scheduling gate. Each scheduling gate must have a unique name field."', args=[d.arg(name='name', type=d.T.string)]),
  withName(name): { name: name },
  '#mixin': 'ignore',
  mixin: self,
}
