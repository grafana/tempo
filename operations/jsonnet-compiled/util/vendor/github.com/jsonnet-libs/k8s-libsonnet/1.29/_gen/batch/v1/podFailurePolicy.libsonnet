{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='podFailurePolicy', url='', help='"PodFailurePolicy describes how failed pods influence the backoffLimit."'),
  '#withRules':: d.fn(help='"A list of pod failure policy rules. The rules are evaluated in order. Once a rule matches a Pod failure, the remaining of the rules are ignored. When no rule matches the Pod failure, the default handling applies - the counter of pod failures is incremented and it is checked against the backoffLimit. At most 20 elements are allowed."', args=[d.arg(name='rules', type=d.T.array)]),
  withRules(rules): { rules: if std.isArray(v=rules) then rules else [rules] },
  '#withRulesMixin':: d.fn(help='"A list of pod failure policy rules. The rules are evaluated in order. Once a rule matches a Pod failure, the remaining of the rules are ignored. When no rule matches the Pod failure, the default handling applies - the counter of pod failures is incremented and it is checked against the backoffLimit. At most 20 elements are allowed."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='rules', type=d.T.array)]),
  withRulesMixin(rules): { rules+: if std.isArray(v=rules) then rules else [rules] },
  '#mixin': 'ignore',
  mixin: self,
}
