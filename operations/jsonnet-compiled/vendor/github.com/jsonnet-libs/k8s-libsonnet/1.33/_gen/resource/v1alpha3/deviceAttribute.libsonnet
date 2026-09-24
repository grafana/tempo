{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='deviceAttribute', url='', help='"DeviceAttribute must have exactly one field set."'),
  '#withBool':: d.fn(help='"BoolValue is a true/false value."', args=[d.arg(name='bool', type=d.T.boolean)]),
  withBool(bool): { bool: bool },
  '#withInt':: d.fn(help='"IntValue is a number."', args=[d.arg(name='int', type=d.T.integer)]),
  withInt(int): { int: int },
  '#withString':: d.fn(help='"StringValue is a string. Must not be longer than 64 characters."', args=[d.arg(name='string', type=d.T.string)]),
  withString(string): { string: string },
  '#withVersion':: d.fn(help='"VersionValue is a semantic version according to semver.org spec 2.0.0. Must not be longer than 64 characters."', args=[d.arg(name='version', type=d.T.string)]),
  withVersion(version): { version: version },
  '#mixin': 'ignore',
  mixin: self,
}
