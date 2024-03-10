{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='validatingAdmissionPolicyStatus', url='', help='"ValidatingAdmissionPolicyStatus represents the status of a ValidatingAdmissionPolicy."'),
  '#typeChecking':: d.obj(help='"TypeChecking contains results of type checking the expressions in the ValidatingAdmissionPolicy"'),
  typeChecking: {
    '#withExpressionWarnings':: d.fn(help='"The type checking warnings for each expression."', args=[d.arg(name='expressionWarnings', type=d.T.array)]),
    withExpressionWarnings(expressionWarnings): { typeChecking+: { expressionWarnings: if std.isArray(v=expressionWarnings) then expressionWarnings else [expressionWarnings] } },
    '#withExpressionWarningsMixin':: d.fn(help='"The type checking warnings for each expression."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='expressionWarnings', type=d.T.array)]),
    withExpressionWarningsMixin(expressionWarnings): { typeChecking+: { expressionWarnings+: if std.isArray(v=expressionWarnings) then expressionWarnings else [expressionWarnings] } },
  },
  '#withConditions':: d.fn(help="\"The conditions represent the latest available observations of a policy's current state.\"", args=[d.arg(name='conditions', type=d.T.array)]),
  withConditions(conditions): { conditions: if std.isArray(v=conditions) then conditions else [conditions] },
  '#withConditionsMixin':: d.fn(help="\"The conditions represent the latest available observations of a policy's current state.\"\n\n**Note:** This function appends passed data to existing values", args=[d.arg(name='conditions', type=d.T.array)]),
  withConditionsMixin(conditions): { conditions+: if std.isArray(v=conditions) then conditions else [conditions] },
  '#withObservedGeneration':: d.fn(help='"The generation observed by the controller."', args=[d.arg(name='observedGeneration', type=d.T.integer)]),
  withObservedGeneration(observedGeneration): { observedGeneration: observedGeneration },
  '#mixin': 'ignore',
  mixin: self,
}
