{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='variable', url='', help='"Variable is the definition of a variable that is used for composition."'),
  '#withExpression':: d.fn(help='"Expression is the expression that will be evaluated as the value of the variable. The CEL expression has access to the same identifiers as the CEL expressions in Validation."', args=[d.arg(name='expression', type=d.T.string)]),
  withExpression(expression): { expression: expression },
  '#withName':: d.fn(help='"Name is the name of the variable. The name must be a valid CEL identifier and unique among all variables. The variable can be accessed in other expressions through `variables` For example, if name is \\"foo\\", the variable will be available as `variables.foo`"', args=[d.arg(name='name', type=d.T.string)]),
  withName(name): { name: name },
  '#mixin': 'ignore',
  mixin: self,
}
