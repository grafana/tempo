{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='v1alpha1', url='', help=''),
  clusterTriggerAuthentication: (import 'clusterTriggerAuthentication.libsonnet'),
  scaledJob: (import 'scaledJob.libsonnet'),
  scaledObject: (import 'scaledObject.libsonnet'),
  triggerAuthentication: (import 'triggerAuthentication.libsonnet'),
}
