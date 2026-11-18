{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='ipBlock', url='', help="\"IPBlock describes a particular CIDR (Ex. \\\"192.168.1.0/24\\\",\\\"2001:db8::/64\\\") that is allowed to the pods matched by a NetworkPolicySpec's podSelector. The except entry describes CIDRs that should not be included within this rule.\""),
  '#withCidr':: d.fn(help='"cidr is a string representing the IPBlock Valid examples are \\"192.168.1.0/24\\" or \\"2001:db8::/64\\', args=[d.arg(name='cidr', type=d.T.string)]),
  withCidr(cidr): { cidr: cidr },
  '#withExcept':: d.fn(help='"except is a slice of CIDRs that should not be included within an IPBlock Valid examples are \\"192.168.1.0/24\\" or \\"2001:db8::/64\\" Except values will be rejected if they are outside the cidr range"', args=[d.arg(name='except', type=d.T.array)]),
  withExcept(except): { except: if std.isArray(v=except) then except else [except] },
  '#withExceptMixin':: d.fn(help='"except is a slice of CIDRs that should not be included within an IPBlock Valid examples are \\"192.168.1.0/24\\" or \\"2001:db8::/64\\" Except values will be rejected if they are outside the cidr range"\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='except', type=d.T.array)]),
  withExceptMixin(except): { except+: if std.isArray(v=except) then except else [except] },
  '#mixin': 'ignore',
  mixin: self,
}
