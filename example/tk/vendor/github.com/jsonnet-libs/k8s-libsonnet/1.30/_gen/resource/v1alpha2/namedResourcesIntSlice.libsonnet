{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='namedResourcesIntSlice', url='', help='"NamedResourcesIntSlice contains a slice of 64-bit integers."'),
  '#withInts':: d.fn(help='"Ints is the slice of 64-bit integers."', args=[d.arg(name='ints', type=d.T.array)]),
  withInts(ints): { ints: if std.isArray(v=ints) then ints else [ints] },
  '#withIntsMixin':: d.fn(help='"Ints is the slice of 64-bit integers."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='ints', type=d.T.array)]),
  withIntsMixin(ints): { ints+: if std.isArray(v=ints) then ints else [ints] },
  '#mixin': 'ignore',
  mixin: self,
}
