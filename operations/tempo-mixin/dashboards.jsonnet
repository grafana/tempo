local mixin = import 'mixin.libsonnet';
local config = import 'config.libsonnet';

config {
  [name]: std.manifestJsonEx(mixin.grafanaDashboards[name], ' ')
  for name in std.objectFields(mixin.grafanaDashboards)
}
