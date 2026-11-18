{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='serviceBackendPort', url='', help='"ServiceBackendPort is the service port being referenced."'),
  '#withName':: d.fn(help='"name is the name of the port on the Service. This is a mutually exclusive setting with \\"Number\\"."', args=[d.arg(name='name', type=d.T.string)]),
  withName(name): { name: name },
  '#withNumber':: d.fn(help='"number is the numerical port number (e.g. 80) on the Service. This is a mutually exclusive setting with \\"Name\\"."', args=[d.arg(name='number', type=d.T.integer)]),
  withNumber(number): { number: number },
  '#mixin': 'ignore',
  mixin: self,
}
