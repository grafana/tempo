{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='podOS', url='', help='"PodOS defines the OS parameters of a pod."'),
  '#withName':: d.fn(help='"Name is the name of the operating system. The currently supported values are linux and windows. Additional value may be defined in future and can be one of: https://github.com/opencontainers/runtime-spec/blob/master/config.md#platform-specific-configuration Clients should expect to handle additional values and treat unrecognized values in this field as os: null"', args=[d.arg(name='name', type=d.T.string)]),
  withName(name): { name: name },
  '#mixin': 'ignore',
  mixin: self,
}
