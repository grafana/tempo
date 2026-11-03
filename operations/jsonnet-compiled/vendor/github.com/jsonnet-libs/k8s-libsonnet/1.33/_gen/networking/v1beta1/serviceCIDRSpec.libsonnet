{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='serviceCIDRSpec', url='', help='"ServiceCIDRSpec define the CIDRs the user wants to use for allocating ClusterIPs for Services."'),
  '#withCidrs':: d.fn(help='"CIDRs defines the IP blocks in CIDR notation (e.g. \\"192.168.0.0/24\\" or \\"2001:db8::/64\\") from which to assign service cluster IPs. Max of two CIDRs is allowed, one of each IP family. This field is immutable."', args=[d.arg(name='cidrs', type=d.T.array)]),
  withCidrs(cidrs): { cidrs: if std.isArray(v=cidrs) then cidrs else [cidrs] },
  '#withCidrsMixin':: d.fn(help='"CIDRs defines the IP blocks in CIDR notation (e.g. \\"192.168.0.0/24\\" or \\"2001:db8::/64\\") from which to assign service cluster IPs. Max of two CIDRs is allowed, one of each IP family. This field is immutable."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='cidrs', type=d.T.array)]),
  withCidrsMixin(cidrs): { cidrs+: if std.isArray(v=cidrs) then cidrs else [cidrs] },
  '#mixin': 'ignore',
  mixin: self,
}
