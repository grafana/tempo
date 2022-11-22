local grr = import 'grizzly/grizzly.libsonnet';
local mixin = import 'mixin.libsonnet';

local grrDashboards = [
  grr.dashboard.new(mixin.grafanaDashboards[fname].uid, mixin.grafanaDashboards[fname])
  for fname in std.objectFields(mixin.grafanaDashboards)
];

// local grrRules = std.map(function(g) grr.rule_group.new(prepared.preparedRules.namespace, g.name, g), prepared.preparedRules.groups + prepared.preparedAlerts.groups);

local grrRules = {};

{
  dashboards: grrDashboards,
  prometheus_rule_groups: grrRules,
}
