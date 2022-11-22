local util = import 'util.libsonnet';
local kind = 'PrometheusRuleGroup';
local recordingRules = 'prometheusRules';
local alertRules = 'prometheusAlerts';
local resource = import 'resource.libsonnet';

{
  getMixinRuleNames(mixins)::
    local flatMixins = [mixins[key] for key in std.objectFieldsAll(mixins)];
    local mixinRules = std.flattenArrays([mixin.prometheusRules.groups for mixin in flatMixins if std.objectHasAll(mixin, recordingRules)]);
    local mixinAlerts = std.flattenArrays([mixin.prometheusAlerts.groups for mixin in flatMixins if std.objectHasAll(mixin, alertRules)]);
    [group.name for group in mixinAlerts] + [group.name for group in mixinRules],

  fromMaps(rules):: { [k]: util.makeResource(kind, k, { groups: rules }, {}) for k in std.objectFields(rules) },

  fromMapsFiltered(rules, excludes):: {
    local filterRules(rules, exclude_list) = [rule for rule in rules.groups if !std.member(exclude_list, rule.name)],
    [k]: util.makeResource(
      kind,
      std.strReplace(std.strReplace(std.strReplace(k, '.json', ''), '.yaml', ''), '.yml', ''),
      { groups: filterRules(rules, excludes) },
      {}
    )
    for k in std.objectFields(rules)
  },

  fromMixins(mixins):: {
    [if std.objectHasAll(mixins[key], alertRules) || std.objectHasAll(mixins[key], recordingRules) then key else null]:
      util.makeResource(
        kind,
        std.strReplace(std.strReplace(std.strReplace(key, '.json', ''), '.yaml', ''), '.yml', ''),
        (if std.objectHasAll(mixins[key], alertRules) then mixins[key].prometheusAlerts else {})
        + (if std.objectHasAll(mixins[key], recordingRules) then mixins[key].prometheusRules else {})
      )
    for key in std.objectFields(mixins)
  },

  rule_group: {
    new(namespace, name, group)::
      resource.new('PrometheusRuleGroup', name)
      + resource.addMetadata('namespace', namespace)
      + resource.withSpec(group),
  },
}
