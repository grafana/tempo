{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='typeChecking', url='', help='"TypeChecking contains results of type checking the expressions in the ValidatingAdmissionPolicy"'),
  '#withExpressionWarnings':: d.fn(help='"The type checking warnings for each expression."', args=[d.arg(name='expressionWarnings', type=d.T.array)]),
  withExpressionWarnings(expressionWarnings): { expressionWarnings: if std.isArray(v=expressionWarnings) then expressionWarnings else [expressionWarnings] },
  '#withExpressionWarningsMixin':: d.fn(help='"The type checking warnings for each expression."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='expressionWarnings', type=d.T.array)]),
  withExpressionWarningsMixin(expressionWarnings): { expressionWarnings+: if std.isArray(v=expressionWarnings) then expressionWarnings else [expressionWarnings] },
  '#mixin': 'ignore',
  mixin: self,
}
