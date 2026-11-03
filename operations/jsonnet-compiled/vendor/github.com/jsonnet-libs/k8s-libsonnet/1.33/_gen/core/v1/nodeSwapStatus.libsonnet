{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='nodeSwapStatus', url='', help='"NodeSwapStatus represents swap memory information."'),
  '#withCapacity':: d.fn(help='"Total amount of swap memory in bytes."', args=[d.arg(name='capacity', type=d.T.integer)]),
  withCapacity(capacity): { capacity: capacity },
  '#mixin': 'ignore',
  mixin: self,
}
