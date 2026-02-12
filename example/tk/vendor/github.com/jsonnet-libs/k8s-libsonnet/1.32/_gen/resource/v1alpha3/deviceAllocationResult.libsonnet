{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='deviceAllocationResult', url='', help='"DeviceAllocationResult is the result of allocating devices."'),
  '#withConfig':: d.fn(help='"This field is a combination of all the claim and class configuration parameters. Drivers can distinguish between those based on a flag.\\n\\nThis includes configuration parameters for drivers which have no allocated devices in the result because it is up to the drivers which configuration parameters they support. They can silently ignore unknown configuration parameters."', args=[d.arg(name='config', type=d.T.array)]),
  withConfig(config): { config: if std.isArray(v=config) then config else [config] },
  '#withConfigMixin':: d.fn(help='"This field is a combination of all the claim and class configuration parameters. Drivers can distinguish between those based on a flag.\\n\\nThis includes configuration parameters for drivers which have no allocated devices in the result because it is up to the drivers which configuration parameters they support. They can silently ignore unknown configuration parameters."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='config', type=d.T.array)]),
  withConfigMixin(config): { config+: if std.isArray(v=config) then config else [config] },
  '#withResults':: d.fn(help='"Results lists all allocated devices."', args=[d.arg(name='results', type=d.T.array)]),
  withResults(results): { results: if std.isArray(v=results) then results else [results] },
  '#withResultsMixin':: d.fn(help='"Results lists all allocated devices."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='results', type=d.T.array)]),
  withResultsMixin(results): { results+: if std.isArray(v=results) then results else [results] },
  '#mixin': 'ignore',
  mixin: self,
}
