{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='successPolicy', url='', help='"SuccessPolicy describes when a Job can be declared as succeeded based on the success of some indexes."'),
  '#withRules':: d.fn(help='"rules represents the list of alternative rules for the declaring the Jobs as successful before `.status.succeeded >= .spec.completions`. Once any of the rules are met, the \\"SucceededCriteriaMet\\" condition is added, and the lingering pods are removed. The terminal state for such a Job has the \\"Complete\\" condition. Additionally, these rules are evaluated in order; Once the Job meets one of the rules, other rules are ignored. At most 20 elements are allowed."', args=[d.arg(name='rules', type=d.T.array)]),
  withRules(rules): { rules: if std.isArray(v=rules) then rules else [rules] },
  '#withRulesMixin':: d.fn(help='"rules represents the list of alternative rules for the declaring the Jobs as successful before `.status.succeeded >= .spec.completions`. Once any of the rules are met, the \\"SucceededCriteriaMet\\" condition is added, and the lingering pods are removed. The terminal state for such a Job has the \\"Complete\\" condition. Additionally, these rules are evaluated in order; Once the Job meets one of the rules, other rules are ignored. At most 20 elements are allowed."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='rules', type=d.T.array)]),
  withRulesMixin(rules): { rules+: if std.isArray(v=rules) then rules else [rules] },
  '#mixin': 'ignore',
  mixin: self,
}
