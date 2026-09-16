{
  local d = (import 'doc-util/main.libsonnet'),
  '#':: d.pkg(name='deviceClassSpec', url='', help='"DeviceClassSpec is used in a [DeviceClass] to define what can be allocated and how to configure it."'),
  '#withConfig':: d.fn(help='"Config defines configuration parameters that apply to each device that is claimed via this class. Some classses may potentially be satisfied by multiple drivers, so each instance of a vendor configuration applies to exactly one driver.\\n\\nThey are passed to the driver, but are not considered while allocating the claim."', args=[d.arg(name='config', type=d.T.array)]),
  withConfig(config): { config: if std.isArray(v=config) then config else [config] },
  '#withConfigMixin':: d.fn(help='"Config defines configuration parameters that apply to each device that is claimed via this class. Some classses may potentially be satisfied by multiple drivers, so each instance of a vendor configuration applies to exactly one driver.\\n\\nThey are passed to the driver, but are not considered while allocating the claim."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='config', type=d.T.array)]),
  withConfigMixin(config): { config+: if std.isArray(v=config) then config else [config] },
  '#withSelectors':: d.fn(help='"Each selector must be satisfied by a device which is claimed via this class."', args=[d.arg(name='selectors', type=d.T.array)]),
  withSelectors(selectors): { selectors: if std.isArray(v=selectors) then selectors else [selectors] },
  '#withSelectorsMixin':: d.fn(help='"Each selector must be satisfied by a device which is claimed via this class."\n\n**Note:** This function appends passed data to existing values', args=[d.arg(name='selectors', type=d.T.array)]),
  withSelectorsMixin(selectors): { selectors+: if std.isArray(v=selectors) then selectors else [selectors] },
  '#mixin': 'ignore',
  mixin: self,
}
