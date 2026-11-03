{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='podFailurePolicyOnPodConditionsPattern', url='', help='"PodFailurePolicyOnPodConditionsPattern describes a pattern for matching an actual pod condition type."'),
  '#withType':: d.fn(help='"Specifies the required Pod condition type. To match a pod condition it is required that specified type equals the pod condition type."', args=[d.arg(name='type', type=d.T.string)]),
  withType(type): { type: type },
  '#mixin': 'ignore',
  mixin: self,
}
