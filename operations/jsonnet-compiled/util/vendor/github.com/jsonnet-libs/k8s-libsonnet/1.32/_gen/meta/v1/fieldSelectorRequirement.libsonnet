{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='fieldSelectorRequirement', url='', help='"FieldSelectorRequirement is a selector that contains values, a key, and an operator that relates the key and values."'),
  '#withKey':: d.fn(help='"key is the field selector key that the requirement applies to."', args=[d.arg(name='key', type=d.T.string)]),
  withKey(key): { key: key },
  '#withOperator':: d.fn(help="\"operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. The list of operators may grow in the future.\"", args=[d.arg(name='operator', type=d.T.string)]),
  withOperator(operator): { operator: operator },
  '#withValues':: d.fn(help='"values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty."', args=[d.arg(name='values', type=d.T.array)]),
  withValues(values): { values: if std.isArray(v=values) then values else [values] },
  '#withValuesMixin':: d.fn(help='"values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='values', type=d.T.array)]),
  withValuesMixin(values): { values+: if std.isArray(v=values) then values else [values] },
  '#mixin': 'ignore',
  mixin: self,
}
