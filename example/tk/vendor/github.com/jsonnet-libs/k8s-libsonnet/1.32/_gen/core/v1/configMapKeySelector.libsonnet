{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='configMapKeySelector', url='', help='"Selects a key from a ConfigMap."'),
  '#withKey':: d.fn(help='"The key to select."', args=[d.arg(name='key', type=d.T.string)]),
  withKey(key): { key: key },
  '#withName':: d.fn(help='"Name of the referent. This field is effectively required, but due to backwards compatibility is allowed to be empty. Instances of this type with an empty value here are almost certainly wrong. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names"', args=[d.arg(name='name', type=d.T.string)]),
  withName(name): { name: name },
  '#withOptional':: d.fn(help='"Specify whether the ConfigMap or its key must be defined"', args=[d.arg(name='optional', type=d.T.boolean)]),
  withOptional(optional): { optional: optional },
  '#mixin': 'ignore',
  mixin: self,
}
