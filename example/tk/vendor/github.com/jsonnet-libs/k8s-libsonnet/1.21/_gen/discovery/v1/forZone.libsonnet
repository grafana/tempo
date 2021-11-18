{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='forZone', url='', help='ForZone provides information about which zones should consume this endpoint.'),
  '#withName':: d.fn(help='name represents the name of the zone.', args=[d.arg(name='name', type=d.T.string)]),
  withName(name): { name: name },
  '#mixin': 'ignore',
  mixin: self,
}
