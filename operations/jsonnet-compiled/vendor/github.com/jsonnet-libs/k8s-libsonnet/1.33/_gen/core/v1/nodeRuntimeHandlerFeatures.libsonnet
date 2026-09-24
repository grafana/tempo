{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='nodeRuntimeHandlerFeatures', url='', help='"NodeRuntimeHandlerFeatures is a set of features implemented by the runtime handler."'),
  '#withRecursiveReadOnlyMounts':: d.fn(help='"RecursiveReadOnlyMounts is set to true if the runtime handler supports RecursiveReadOnlyMounts."', args=[d.arg(name='recursiveReadOnlyMounts', type=d.T.boolean)]),
  withRecursiveReadOnlyMounts(recursiveReadOnlyMounts): { recursiveReadOnlyMounts: recursiveReadOnlyMounts },
  '#withUserNamespaces':: d.fn(help='"UserNamespaces is set to true if the runtime handler supports UserNamespaces, including for volumes."', args=[d.arg(name='userNamespaces', type=d.T.boolean)]),
  withUserNamespaces(userNamespaces): { userNamespaces: userNamespaces },
  '#mixin': 'ignore',
  mixin: self,
}
