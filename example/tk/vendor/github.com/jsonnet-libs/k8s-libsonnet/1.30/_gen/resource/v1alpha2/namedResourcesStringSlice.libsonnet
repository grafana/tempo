{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='namedResourcesStringSlice', url='', help='"NamedResourcesStringSlice contains a slice of strings."'),
  '#withStrings':: d.fn(help='"Strings is the slice of strings."', args=[d.arg(name='strings', type=d.T.array)]),
  withStrings(strings): { strings: if std.isArray(v=strings) then strings else [strings] },
  '#withStringsMixin':: d.fn(help='"Strings is the slice of strings."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='strings', type=d.T.array)]),
  withStringsMixin(strings): { strings+: if std.isArray(v=strings) then strings else [strings] },
  '#mixin': 'ignore',
  mixin: self,
}
