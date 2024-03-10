{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='paramKind', url='', help='"ParamKind is a tuple of Group Kind and Version."'),
  '#withKind':: d.fn(help='"Kind is the API kind the resources belong to. Required."', args=[d.arg(name='kind', type=d.T.string)]),
  withKind(kind): { kind: kind },
  '#mixin': 'ignore',
  mixin: self,
}
