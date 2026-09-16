{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='nodeFeatures', url='', help='"NodeFeatures describes the set of features implemented by the CRI implementation. The features contained in the NodeFeatures should depend only on the cri implementation independent of runtime handlers."'),
  '#withSupplementalGroupsPolicy':: d.fn(help='"SupplementalGroupsPolicy is set to true if the runtime supports SupplementalGroupsPolicy and ContainerUser."', args=[d.arg(name='supplementalGroupsPolicy', type=d.T.boolean)]),
  withSupplementalGroupsPolicy(supplementalGroupsPolicy): { supplementalGroupsPolicy: supplementalGroupsPolicy },
  '#mixin': 'ignore',
  mixin: self,
}
