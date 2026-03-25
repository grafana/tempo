{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='ingressPortStatus', url='', help='"IngressPortStatus represents the error condition of a service port"'),
  '#withError':: d.fn(help='"error is to record the problem with the service port The format of the error shall comply with the following rules: - built-in error values shall be specified in this file and those shall use\\n  CamelCase names\\n- cloud provider specific error values must have names that comply with the\\n  format foo.example.com/CamelCase."', args=[d.arg(name='err', type=d.T.string)]),
  withError(err): { 'error': err },
  '#withPort':: d.fn(help='"port is the port number of the ingress port."', args=[d.arg(name='port', type=d.T.integer)]),
  withPort(port): { port: port },
  '#withProtocol':: d.fn(help='"protocol is the protocol of the ingress port. The supported values are: \\"TCP\\", \\"UDP\\", \\"SCTP\\', args=[d.arg(name='protocol', type=d.T.string)]),
  withProtocol(protocol): { protocol: protocol },
  '#mixin': 'ignore',
  mixin: self,
}
