{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='podIP', url='', help='"PodIP represents a single IP address allocated to the pod."'),
  '#withIp':: d.fn(help='"IP is the IP address assigned to the pod"', args=[d.arg(name='ip', type=d.T.string)]),
  withIp(ip): { ip: ip },
  '#mixin': 'ignore',
  mixin: self,
}
