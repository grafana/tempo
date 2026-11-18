{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='expressionWarning', url='', help='"ExpressionWarning is a warning information that targets a specific expression."'),
  '#withFieldRef':: d.fn(help='"The path to the field that refers the expression. For example, the reference to the expression of the first item of validations is \\"spec.validations[0].expression\\', args=[d.arg(name='fieldRef', type=d.T.string)]),
  withFieldRef(fieldRef): { fieldRef: fieldRef },
  '#withWarning':: d.fn(help='"The content of type checking information in a human-readable form. Each line of the warning contains the type that the expression is checked against, followed by the type check error from the compiler."', args=[d.arg(name='warning', type=d.T.string)]),
  withWarning(warning): { warning: warning },
  '#mixin': 'ignore',
  mixin: self,
}
