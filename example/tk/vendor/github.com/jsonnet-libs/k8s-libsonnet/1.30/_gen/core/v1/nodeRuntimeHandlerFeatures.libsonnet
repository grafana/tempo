{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='nodeRuntimeHandlerFeatures', url='', help='"NodeRuntimeHandlerFeatures is a set of runtime features."'),
  '#withRecursiveReadOnlyMounts':: d.fn(help='"RecursiveReadOnlyMounts is set to true if the runtime handler supports RecursiveReadOnlyMounts."', args=[d.arg(name='recursiveReadOnlyMounts', type=d.T.boolean)]),
  withRecursiveReadOnlyMounts(recursiveReadOnlyMounts): { recursiveReadOnlyMounts: recursiveReadOnlyMounts },
  '#mixin': 'ignore',
  mixin: self,
}
