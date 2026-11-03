{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='envFromSource', url='', help='"EnvFromSource represents the source of a set of ConfigMaps or Secrets"'),
  '#configMapRef':: d.obj(help="\"ConfigMapEnvSource selects a ConfigMap to populate the environment variables with.\\n\\nThe contents of the target ConfigMap's Data field will represent the key-value pairs as environment variables.\""),
  configMapRef: {
    '#withName':: d.fn(help='"Name of the referent. This field is effectively required, but due to backwards compatibility is allowed to be empty. Instances of this type with an empty value here are almost certainly wrong. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names"', args=[d.arg(name='name', type=d.T.string)]),
    withName(name): { configMapRef+: { name: name } },
    '#withOptional':: d.fn(help='"Specify whether the ConfigMap must be defined"', args=[d.arg(name='optional', type=d.T.boolean)]),
    withOptional(optional): { configMapRef+: { optional: optional } },
  },
  '#secretRef':: d.obj(help="\"SecretEnvSource selects a Secret to populate the environment variables with.\\n\\nThe contents of the target Secret's Data field will represent the key-value pairs as environment variables.\""),
  secretRef: {
    '#withName':: d.fn(help='"Name of the referent. This field is effectively required, but due to backwards compatibility is allowed to be empty. Instances of this type with an empty value here are almost certainly wrong. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names"', args=[d.arg(name='name', type=d.T.string)]),
    withName(name): { secretRef+: { name: name } },
    '#withOptional':: d.fn(help='"Specify whether the Secret must be defined"', args=[d.arg(name='optional', type=d.T.boolean)]),
    withOptional(optional): { secretRef+: { optional: optional } },
  },
  '#withPrefix':: d.fn(help='"Optional text to prepend to the name of each environment variable. Must be a C_IDENTIFIER."', args=[d.arg(name='prefix', type=d.T.string)]),
  withPrefix(prefix): { prefix: prefix },
  '#mixin': 'ignore',
  mixin: self,
}
