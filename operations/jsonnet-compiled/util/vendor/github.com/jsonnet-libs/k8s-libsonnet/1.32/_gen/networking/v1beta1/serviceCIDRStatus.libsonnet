{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='serviceCIDRStatus', url='', help='"ServiceCIDRStatus describes the current state of the ServiceCIDR."'),
  '#withConditions':: d.fn(help='"conditions holds an array of metav1.Condition that describe the state of the ServiceCIDR. Current service state"', args=[d.arg(name='conditions', type=d.T.array)]),
  withConditions(conditions): { conditions: if std.isArray(v=conditions) then conditions else [conditions] },
  '#withConditionsMixin':: d.fn(help='"conditions holds an array of metav1.Condition that describe the state of the ServiceCIDR. Current service state"\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='conditions', type=d.T.array)]),
  withConditionsMixin(conditions): { conditions+: if std.isArray(v=conditions) then conditions else [conditions] },
  '#mixin': 'ignore',
  mixin: self,
}
