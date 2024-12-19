{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='namedResourcesResources', url='', help='"NamedResourcesResources is used in ResourceModel."'),
  '#withInstances':: d.fn(help='"The list of all individual resources instances currently available."', args=[d.arg(name='instances', type=d.T.array)]),
  withInstances(instances): { instances: if std.isArray(v=instances) then instances else [instances] },
  '#withInstancesMixin':: d.fn(help='"The list of all individual resources instances currently available."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='instances', type=d.T.array)]),
  withInstancesMixin(instances): { instances+: if std.isArray(v=instances) then instances else [instances] },
  '#mixin': 'ignore',
  mixin: self,
}
