{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='hostIP', url='', help='"HostIP represents a single IP address allocated to the host."'),
  '#withIp':: d.fn(help='"IP is the IP address assigned to the host"', args=[d.arg(name='ip', type=d.T.string)]),
  withIp(ip): { ip: ip },
  '#mixin': 'ignore',
  mixin: self,
}
