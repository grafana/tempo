{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='ingressLoadBalancerIngress', url='', help='"IngressLoadBalancerIngress represents the status of a load-balancer ingress point."'),
  '#withHostname':: d.fn(help='"hostname is set for load-balancer ingress points that are DNS based."', args=[d.arg(name='hostname', type=d.T.string)]),
  withHostname(hostname): { hostname: hostname },
  '#withIp':: d.fn(help='"ip is set for load-balancer ingress points that are IP based."', args=[d.arg(name='ip', type=d.T.string)]),
  withIp(ip): { ip: ip },
  '#withPorts':: d.fn(help='"ports provides information about the ports exposed by this LoadBalancer."', args=[d.arg(name='ports', type=d.T.array)]),
  withPorts(ports): { ports: if std.isArray(v=ports) then ports else [ports] },
  '#withPortsMixin':: d.fn(help='"ports provides information about the ports exposed by this LoadBalancer."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='ports', type=d.T.array)]),
  withPortsMixin(ports): { ports+: if std.isArray(v=ports) then ports else [ports] },
  '#mixin': 'ignore',
  mixin: self,
}
