{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='nodeRuntimeHandler', url='', help='"NodeRuntimeHandler is a set of runtime handler information."'),
  '#features':: d.obj(help='"NodeRuntimeHandlerFeatures is a set of features implemented by the runtime handler."'),
  features: {
    '#withRecursiveReadOnlyMounts':: d.fn(help='"RecursiveReadOnlyMounts is set to true if the runtime handler supports RecursiveReadOnlyMounts."', args=[d.arg(name='recursiveReadOnlyMounts', type=d.T.boolean)]),
    withRecursiveReadOnlyMounts(recursiveReadOnlyMounts): { features+: { recursiveReadOnlyMounts: recursiveReadOnlyMounts } },
    '#withUserNamespaces':: d.fn(help='"UserNamespaces is set to true if the runtime handler supports UserNamespaces, including for volumes."', args=[d.arg(name='userNamespaces', type=d.T.boolean)]),
    withUserNamespaces(userNamespaces): { features+: { userNamespaces: userNamespaces } },
  },
  '#withName':: d.fn(help='"Runtime handler name. Empty for the default runtime handler."', args=[d.arg(name='name', type=d.T.string)]),
  withName(name): { name: name },
  '#mixin': 'ignore',
  mixin: self,
}
