{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='appArmorProfile', url='', help="\"AppArmorProfile defines a pod or container's AppArmor settings.\""),
  '#withLocalhostProfile':: d.fn(help='"localhostProfile indicates a profile loaded on the node that should be used. The profile must be preconfigured on the node to work. Must match the loaded name of the profile. Must be set if and only if type is \\"Localhost\\"."', args=[d.arg(name='localhostProfile', type=d.T.string)]),
  withLocalhostProfile(localhostProfile): { localhostProfile: localhostProfile },
  '#withType':: d.fn(help="\"type indicates which kind of AppArmor profile will be applied. Valid options are:\\n  Localhost - a profile pre-loaded on the node.\\n  RuntimeDefault - the container runtime's default profile.\\n  Unconfined - no AppArmor enforcement.\"", args=[d.arg(name='type', type=d.T.string)]),
  withType(type): { type: type },
  '#mixin': 'ignore',
  mixin: self,
}
