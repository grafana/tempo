{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='deviceTaint', url='', help='"The device this taint is attached to has the \\"effect\\" on any claim which does not tolerate the taint and, through the claim, to pods using the claim."'),
  '#withEffect':: d.fn(help='"The effect of the taint on claims that do not tolerate the taint and through such claims on the pods using them. Valid effects are NoSchedule and NoExecute. PreferNoSchedule as used for nodes is not valid here."', args=[d.arg(name='effect', type=d.T.string)]),
  withEffect(effect): { effect: effect },
  '#withKey':: d.fn(help='"The taint key to be applied to a device. Must be a label name."', args=[d.arg(name='key', type=d.T.string)]),
  withKey(key): { key: key },
  '#withTimeAdded':: d.fn(help='"Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers."', args=[d.arg(name='timeAdded', type=d.T.string)]),
  withTimeAdded(timeAdded): { timeAdded: timeAdded },
  '#withValue':: d.fn(help='"The taint value corresponding to the taint key. Must be a label value."', args=[d.arg(name='value', type=d.T.string)]),
  withValue(value): { value: value },
  '#mixin': 'ignore',
  mixin: self,
}
