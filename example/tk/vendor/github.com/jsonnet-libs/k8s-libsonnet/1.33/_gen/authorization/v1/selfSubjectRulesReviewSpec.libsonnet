{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='selfSubjectRulesReviewSpec', url='', help='"SelfSubjectRulesReviewSpec defines the specification for SelfSubjectRulesReview."'),
  '#withNamespace':: d.fn(help='"Namespace to evaluate rules for. Required."', args=[d.arg(name='namespace', type=d.T.string)]),
  withNamespace(namespace): { namespace: namespace },
  '#mixin': 'ignore',
  mixin: self,
}
