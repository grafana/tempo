{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='storageVersionMigrationStatus', url='', help='"Status of the storage version migration."'),
  '#withConditions':: d.fn(help="\"The latest available observations of the migration's current state.\"", args=[d.arg(name='conditions', type=d.T.array)]),
  withConditions(conditions): { conditions: if std.isArray(v=conditions) then conditions else [conditions] },
  '#withConditionsMixin':: d.fn(help="\"The latest available observations of the migration's current state.\"\n\n**Note:** This function appends passed data to existing values", args=[d.arg(name='conditions', type=d.T.array)]),
  withConditionsMixin(conditions): { conditions+: if std.isArray(v=conditions) then conditions else [conditions] },
  '#withResourceVersion':: d.fn(help='"ResourceVersion to compare with the GC cache for performing the migration. This is the current resource version of given group, version and resource when kube-controller-manager first observes this StorageVersionMigration resource."', args=[d.arg(name='resourceVersion', type=d.T.string)]),
  withResourceVersion(resourceVersion): { resourceVersion: resourceVersion },
  '#mixin': 'ignore',
  mixin: self,
}
