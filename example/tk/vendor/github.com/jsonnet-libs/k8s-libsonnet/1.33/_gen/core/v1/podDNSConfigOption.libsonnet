{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='podDNSConfigOption', url='', help='"PodDNSConfigOption defines DNS resolver options of a pod."'),
  '#withName':: d.fn(help="\"Name is this DNS resolver option's name. Required.\"", args=[d.arg(name='name', type=d.T.string)]),
  withName(name): { name: name },
  '#withValue':: d.fn(help="\"Value is this DNS resolver option's value.\"", args=[d.arg(name='value', type=d.T.string)]),
  withValue(value): { value: value },
  '#mixin': 'ignore',
  mixin: self,
}
