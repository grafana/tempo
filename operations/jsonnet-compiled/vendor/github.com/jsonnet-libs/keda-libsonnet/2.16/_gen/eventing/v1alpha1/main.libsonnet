{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='v1alpha1', url='', help=''),
  cloudEventSource: (import 'cloudEventSource.libsonnet'),
  clusterCloudEventSource: (import 'clusterCloudEventSource.libsonnet'),
}
