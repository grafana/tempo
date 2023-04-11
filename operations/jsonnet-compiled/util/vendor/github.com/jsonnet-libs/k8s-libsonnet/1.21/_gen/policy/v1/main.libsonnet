{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='v1', url='', help=''),
  podDisruptionBudget: (import 'podDisruptionBudget.libsonnet'),
  podDisruptionBudgetSpec: (import 'podDisruptionBudgetSpec.libsonnet'),
  podDisruptionBudgetStatus: (import 'podDisruptionBudgetStatus.libsonnet'),
}
