{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='storageVersionMigrationSpec', url='', help='"Spec of the storage version migration."'),
  '#resource':: d.obj(help='"The names of the group, the version, and the resource."'),
  resource: {
    '#withGroup':: d.fn(help='"The name of the group."', args=[d.arg(name='group', type=d.T.string)]),
    withGroup(group): { resource+: { group: group } },
    '#withResource':: d.fn(help='"The name of the resource."', args=[d.arg(name='resource', type=d.T.string)]),
    withResource(resource): { resource+: { resource: resource } },
    '#withVersion':: d.fn(help='"The name of the version."', args=[d.arg(name='version', type=d.T.string)]),
    withVersion(version): { resource+: { version: version } },
  },
  '#withContinueToken':: d.fn(help='"The token used in the list options to get the next chunk of objects to migrate. When the .status.conditions indicates the migration is \\"Running\\", users can use this token to check the progress of the migration."', args=[d.arg(name='continueToken', type=d.T.string)]),
  withContinueToken(continueToken): { continueToken: continueToken },
  '#mixin': 'ignore',
  mixin: self,
}
