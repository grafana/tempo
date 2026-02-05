{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='v1alpha1', url='', help=''),
  groupVersionResource: (import 'groupVersionResource.libsonnet'),
  migrationCondition: (import 'migrationCondition.libsonnet'),
  storageVersionMigration: (import 'storageVersionMigration.libsonnet'),
  storageVersionMigrationSpec: (import 'storageVersionMigrationSpec.libsonnet'),
  storageVersionMigrationStatus: (import 'storageVersionMigrationStatus.libsonnet'),
}
