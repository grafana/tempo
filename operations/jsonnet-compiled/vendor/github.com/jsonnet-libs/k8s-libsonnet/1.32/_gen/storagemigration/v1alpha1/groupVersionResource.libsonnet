{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='groupVersionResource', url='', help='"The names of the group, the version, and the resource."'),
  '#withGroup':: d.fn(help='"The name of the group."', args=[d.arg(name='group', type=d.T.string)]),
  withGroup(group): { group: group },
  '#withResource':: d.fn(help='"The name of the resource."', args=[d.arg(name='resource', type=d.T.string)]),
  withResource(resource): { resource: resource },
  '#withVersion':: d.fn(help='"The name of the version."', args=[d.arg(name='version', type=d.T.string)]),
  withVersion(version): { version: version },
  '#mixin': 'ignore',
  mixin: self,
}
