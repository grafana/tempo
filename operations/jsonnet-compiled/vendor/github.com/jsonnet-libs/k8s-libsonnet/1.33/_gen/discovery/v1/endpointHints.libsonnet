{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='endpointHints', url='', help='"EndpointHints provides hints describing how an endpoint should be consumed."'),
  '#withForNodes':: d.fn(help='"forNodes indicates the node(s) this endpoint should be consumed by when using topology aware routing. May contain a maximum of 8 entries. This is an Alpha feature and is only used when the PreferSameTrafficDistribution feature gate is enabled."', args=[d.arg(name='forNodes', type=d.T.array)]),
  withForNodes(forNodes): { forNodes: if std.isArray(v=forNodes) then forNodes else [forNodes] },
  '#withForNodesMixin':: d.fn(help='"forNodes indicates the node(s) this endpoint should be consumed by when using topology aware routing. May contain a maximum of 8 entries. This is an Alpha feature and is only used when the PreferSameTrafficDistribution feature gate is enabled."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='forNodes', type=d.T.array)]),
  withForNodesMixin(forNodes): { forNodes+: if std.isArray(v=forNodes) then forNodes else [forNodes] },
  '#withForZones':: d.fn(help='"forZones indicates the zone(s) this endpoint should be consumed by when using topology aware routing. May contain a maximum of 8 entries."', args=[d.arg(name='forZones', type=d.T.array)]),
  withForZones(forZones): { forZones: if std.isArray(v=forZones) then forZones else [forZones] },
  '#withForZonesMixin':: d.fn(help='"forZones indicates the zone(s) this endpoint should be consumed by when using topology aware routing. May contain a maximum of 8 entries."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='forZones', type=d.T.array)]),
  withForZonesMixin(forZones): { forZones+: if std.isArray(v=forZones) then forZones else [forZones] },
  '#mixin': 'ignore',
  mixin: self,
}
