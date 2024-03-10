{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='crossVersionObjectReference', url='', help='"CrossVersionObjectReference contains enough information to let you identify the referred resource."'),
  '#withKind':: d.fn(help='"kind is the kind of the referent; More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds"', args=[d.arg(name='kind', type=d.T.string)]),
  withKind(kind): { kind: kind },
  '#withName':: d.fn(help='"name is the name of the referent; More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names"', args=[d.arg(name='name', type=d.T.string)]),
  withName(name): { name: name },
  '#mixin': 'ignore',
  mixin: self,
}
