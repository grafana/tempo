{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='endpointHints', url='', help='"EndpointHints provides hints describing how an endpoint should be consumed."'),
  '#withForZones':: d.fn(help='"forZones indicates the zone(s) this endpoint should be consumed by to enable topology aware routing."', args=[d.arg(name='forZones', type=d.T.array)]),
  withForZones(forZones): { forZones: if std.isArray(v=forZones) then forZones else [forZones] },
  '#withForZonesMixin':: d.fn(help='"forZones indicates the zone(s) this endpoint should be consumed by to enable topology aware routing."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='forZones', type=d.T.array)]),
  withForZonesMixin(forZones): { forZones+: if std.isArray(v=forZones) then forZones else [forZones] },
  '#mixin': 'ignore',
  mixin: self,
}
